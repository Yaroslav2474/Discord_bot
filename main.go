package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "modernc.org/sqlite"
)

const XP_MINUTE = 10
const XP_PER_LEVEL = 100

var LVL_Roles = map[int]string{
	5:  "role_id_lvl_5",
	10: "role_id_lvl_10",
	15: "role_id_lvl_15",
	20: "role_id_lvl_20",
	25: "role_id_lvl_25",
}

var db *sql.DB

type Structura struct {
	First string `json:"message"`
}

type UserInf struct {
	UserID   int
	Username string
	Lvl      int
	XP       int
}

type Command struct {
	Keywords []string `json:"keywords"`
	Replay   string   `json:"replay,omitempty"`
	Reaction string   `json:"reaction,omitempty"`
}

type VoiceSession struct {
	SessionID    int
	UserID       string
	GuildID      string
	ChannelID    string
	JoinedTime   time.Time
	LastXPUpdate time.Time
	CurrentLvl   int
}

var Rules []Command
var active_session = map[string]*VoiceSession{}
var session_mutex sync.RWMutex

func request_to_API() (string, error) {
	url := "https://dog.ceo/api/breeds/image/random"

	resp, err := http.Get(url)

	if err != nil {
		fmt.Println("Error when sending the get request: ", err)
		return "", fmt.Errorf("1")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Println("Error when read: ", err)
		return "", fmt.Errorf("1")
	}

	var hz Structura

	err = json.Unmarshal(body, &hz)

	if err != nil {
		fmt.Println("Error when unmarshaling: ", err)
		return "", fmt.Errorf("1")
	}

	img := hz.First

	return img, nil

}

func loadCommands(filename string) error {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, &Rules)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	userMessage := strings.ToLower(m.Content)

	if strings.Contains(userMessage, "2") || strings.Contains(userMessage, "!level") || strings.Contains(userMessage, "!profile") || strings.Contains(userMessage, "!уровень") || strings.Contains(userMessage, "!профиль") {
		lvl, xp, err := getUserLevel(m.Author.ID)
		if err != nil {
			fmt.Printf("[ERROR] Failed to get level for user %s: %v\n", m.Author.ID, err)
			s.ChannelMessageSend(m.ChannelID, "Error: Failed to get your level")
			return
		}

		if lvl == 0 {
			s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" You don't have any voice channel experience yet. Join a voice channel to earn levels.")
			return
		}

		message := fmt.Sprintf("%s\nYour Profile:\nLevel: %d\nExperience: %d / %d", m.Author.Mention(), lvl, xp, XP_PER_LEVEL)

		nextLevel := lvl + 1
		nextRoleID := getRolesIDForLevel(nextLevel)
		if nextRoleID != "" {
			message += fmt.Sprintf("\nNext Level: %d\nReward: <@&%s>", nextLevel, nextRoleID)
		}

		s.ChannelMessageSend(m.ChannelID, message)
		return
	}

	if strings.Contains(userMessage, "хуй") {
		emoji := []string{"%F0%9F%87%AD", "%f0%9f%87%ba", "%F0%9F%87%AE"}
		for _, e := range emoji {
			err := s.MessageReactionAdd(m.ChannelID, m.ID, e)
			if err != nil {
				fmt.Println("Error adding letter reaction: ", err)
			}
		}
		return
	}

	if strings.Contains(userMessage, "funy time") || strings.Contains(userMessage, "поднять настроение") || strings.Contains(userMessage, "1") {
		url_img, err := request_to_API()

		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Failed to execute the function :(")
			return
		}

		s.ChannelMessageSend(m.ChannelID, url_img)
		return
	}

	if strings.Contains(userMessage, "андреич") {
		imagePath := "matvey.png"

		file, err := os.Open(imagePath)
		if err != nil {
			fmt.Println("Error opening image for Matvey: ", err)
			s.ChannelMessageSend(m.ChannelID, "Failed to find image for Matvey")
			return
		}
		defer file.Close()

		msgData := &discordgo.MessageSend{
			Content: "На месте епт",
			Files: []*discordgo.File{
				{
					Name:        "matvey.png",
					ContentType: "image/png",
					Reader:      file,
				},
			},
		}

		_, err = s.ChannelMessageSendComplex(m.ChannelID, msgData)
		if err != nil {
			fmt.Println("Error sending image for Matvey: ", err)
		}
		return
	}

	for _, rule := range Rules {
		for _, keyword := range rule.Keywords {
			if strings.Contains(userMessage, strings.ToLower(keyword)) {
				if rule.Replay != "" {
					s.ChannelMessageSend(m.ChannelID, rule.Replay)
				}
				if rule.Reaction != "" {
					err := s.MessageReactionAdd(m.ChannelID, m.ID, rule.Reaction)
					if err != nil {
						fmt.Println("Error adding reaction: ", err)
					}
				}
				return
			}
		}
	}
}

func updateRoles(ass *discordgo.Session, userID, guildID string, newlvl int) error {
	member, err := ass.GuildMember(guildID, userID)
	if err != nil {
		fmt.Printf("[ERROR] Failed to get guild member %s: %v\n", userID, err)
		return err
	}

	currentRolesMap := make(map[string]bool)
	for _, roleID := range member.Roles {
		currentRolesMap[roleID] = true
	}

	desiredRoles := make(map[string]bool)
	for lvl, roleID := range LVL_Roles {
		if lvl <= newlvl {
			desiredRoles[roleID] = true
		}
	}

	for roleID := range desiredRoles {
		if !currentRolesMap[roleID] {
			err := ass.GuildMemberRoleAdd(guildID, userID, roleID)
			if err != nil {
				fmt.Printf("[ERROR] Failed to add role %s: %v\n", roleID, err)
			} else {
				fmt.Printf("[ROLE+] Role %s added to user (level %d)\n", roleID, newlvl)
			}
		}
	}

	for roleID := range currentRolesMap {
		isLevelRole := false
		for _, levelRoleID := range LVL_Roles {
			if roleID == levelRoleID {
				isLevelRole = true
				break
			}
		}

		if isLevelRole && !desiredRoles[roleID] {
			err := ass.GuildMemberRoleRemove(guildID, userID, roleID)
			if err != nil {
				fmt.Printf("[ERROR] Failed to remove role %s: %v\n", roleID, err)
			} else {
				fmt.Printf("[ROLE-] Role %s removed from user (level %d)\n", roleID, newlvl)
			}
		}
	}

	return nil
}

func initDB() error {
	var err error
	db, err = sql.Open("sqlite", "voise.db")
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec("PRAGMA synchronous = NORMAL;")
	if err != nil {
		return err
	}
	return db.Ping()
}

func createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS voise_sessions(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		userID TEXT,
		guildID TEXT,
		channelID TEXT,
		joinedTime DATETIME,
		leftTime DATETIME,
		duration INTEGER DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS user_xp(
		userID TEXT PRIMARY KEY,
		xp INTEGER DEFAULT 0,
		lvl INTEGER DEFAULT 1
	);
	`
	_, err := db.Exec(query)
	return err
}

func getRolesIDForLevel(lvl int) string {
	if roleID, exists := LVL_Roles[lvl]; exists {
		return roleID
	}
	return ""
}

func addXP(userID string, xp int) (int, int, error) {
	var currentXP, lvl int
	err := db.QueryRow(`
	SELECT xp, lvl FROM user_xp WHERE userID = ?
	`, userID).Scan(&currentXP, &lvl)

	if err == sql.ErrNoRows {
		currentXP = 0
		lvl = 1
		_, err = db.Exec(`
		INSERT INTO user_xp (
			userID, xp, lvl
		) VALUES (
		?,0,1 
		)
		`, userID)
		if err != nil {
			return 0, 0, err
		}
	} else if err != nil {
		return 0, 0, err
	}

	currentXP += xp
	newlvl := lvl

	for currentXP >= XP_PER_LEVEL {
		currentXP -= XP_PER_LEVEL
		newlvl++
	}

	_, err = db.Exec(`
	UPDATE user_xp SET xp = ?, lvl = ? WHERE userID = ?
	`, currentXP, newlvl, userID)

	if err != nil {
		return 0, 0, err
	}

	return newlvl, currentXP, nil
}

func getUserLevel(userID string) (int, int, error) {
	var currentXP, lvl int
	err := db.QueryRow(`
	SELECT xp, lvl FROM user_xp WHERE userID = ?
	`, userID).Scan(&currentXP, &lvl)

	if err == sql.ErrNoRows {
		return 0, 0, nil
	} else if err != nil {
		return 0, 0, err
	}

	return lvl, currentXP, nil
}

func start_voice_session(userID, guildID, channelID string) error {
	now := time.Now()

	currentLvl, _, err := getUserLevel(userID)
	if err != nil {
		fmt.Printf("[ERROR] Failed to get user level for %s: %v\n", userID, err)
		currentLvl = 1
	}

	result, err := db.Exec(`
	INSERT INTO voise_sessions(
		userID ,
		guildID ,
		channelID ,
		joinedTime
	) VALUES (
		?, ?, ?, ?
	);
	`, userID, guildID, channelID, now)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	session_mutex.Lock()
	active_session[userID] = &VoiceSession{
		SessionID:    int(id),
		UserID:       userID,
		GuildID:      guildID,
		ChannelID:    channelID,
		JoinedTime:   now,
		LastXPUpdate: now,
		CurrentLvl:   currentLvl,
	}
	session_mutex.Unlock()

	fmt.Printf("[JOIN] %s -> %s\n", userID, channelID)
	return nil
}

func end_voice_session(ass *discordgo.Session, userID string) error {
	session_mutex.Lock()
	session, exists := active_session[userID]
	session_mutex.Unlock()

	if !exists {
		return nil
	}

	leftAt := time.Now()
	duration := int(leftAt.Sub(session.JoinedTime).Seconds())

	_, err := db.Exec(`
	UPDATE voise_sessions SET leftTime = ?, duration = ? WHERE id = ?
	`, leftAt, duration, session.SessionID)
	if err != nil {
		return err
	}

	session_mutex.Lock()
	delete(active_session, userID)
	session_mutex.Unlock()

	fmt.Printf("[LEAVE] %s duration=%d seconds\n", userID, duration)
	return nil
}

func flushSession(ass *discordgo.Session) {
	session_mutex.RLock()
	userIDs := make([]string, 0, len(active_session))
	for userID := range active_session {
		userIDs = append(userIDs, userID)
	}
	session_mutex.RUnlock()

	for _, userID := range userIDs {
		err := end_voice_session(ass, userID)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func updateXPTicker(ass *discordgo.Session, stopChan chan bool) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()

			session_mutex.RLock()
			userIDs := make([]string, 0, len(active_session))
			for userID := range active_session {
				userIDs = append(userIDs, userID)
			}
			session_mutex.RUnlock()

			for _, userID := range userIDs {
				session_mutex.RLock()
				session, exists := active_session[userID]
				session_mutex.RUnlock()

				if !exists || session == nil {
					continue
				}

				timeSinceUpdate := now.Sub(session.LastXPUpdate).Seconds()
				if timeSinceUpdate < 1 {
					continue
				}

				xpToAdd := int(timeSinceUpdate/60) * XP_MINUTE
				if xpToAdd > 0 {
					lvl, totalXP, err := addXP(userID, xpToAdd)
					if err != nil {
						fmt.Printf("[ERROR] Failed to add XP for user %s: %v\n", userID, err)
						continue
					}

					if lvl > session.CurrentLvl {
						err = updateRoles(ass, userID, session.GuildID, lvl)
						if err != nil {
							fmt.Printf("[ERROR] Failed to update roles for user %s: %v\n", userID, err)
						}
						fmt.Printf("[XP+] %s gained %d XP (%.0fs), NEW LEVEL: %d, total XP: %d\n", userID, xpToAdd, timeSinceUpdate, lvl, totalXP)

						session_mutex.Lock()
						if activeSession, ok := active_session[userID]; ok {
							activeSession.CurrentLvl = lvl
						}
						session_mutex.Unlock()
					} else {
						fmt.Printf("[XP+] %s gained %d XP (%.0fs), total XP: %d\n", userID, xpToAdd, timeSinceUpdate, totalXP)
					}

					session_mutex.Lock()
					if activeSession, ok := active_session[userID]; ok {
						activeSession.LastXPUpdate = now
					}
					session_mutex.Unlock()
				}
			}
		case <-stopChan:
			return
		}
	}
}

func voiceState(ass *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	userID := v.UserID
	guildID := v.GuildID

	session_mutex.RLock()
	current, exists := active_session[userID]
	session_mutex.RUnlock()

	if v.ChannelID == "" {
		if exists {
			err := end_voice_session(ass, userID)
			if err != nil {
				fmt.Println(err)
			}
			return
		}
	}

	if !exists {
		err := start_voice_session(userID, guildID, v.ChannelID)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	if current.ChannelID != v.ChannelID {
		err := end_voice_session(ass, userID)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = start_voice_session(userID, guildID, v.ChannelID)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func main() {
	TokenBot := os.Getenv("DISCORD_TOKEN")
	if TokenBot == "" {
		panic("Error: bot token not initialization")
	}

	err := initDB()

	if err != nil {
		fmt.Println(err)
		return
	}

	err = createTable()

	if err != nil {
		fmt.Println(err)
		return
	}

	err = loadCommands("command.json")
	if err != nil {
		fmt.Println("Error load json: ", err)
		return
	}
	fmt.Println("json loaded successfully")

	bt, err := discordgo.New("Bot " + TokenBot)
	if err != nil {
		fmt.Println("Error creating discord session: ", err)
		return
	}

	bt.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions | discordgo.IntentsGuildVoiceStates | discordgo.IntentsGuilds
	bt.AddHandler(messageCreate)
	bt.AddHandler(voiceState)

	err = bt.Open()
	if err != nil {
		fmt.Println("Error opening connection: ", err)
		return
	}

	fmt.Println("Bot is now running. Press CTRL-C to exit.")

	stopXPTicker := make(chan bool)
	go updateXPTicker(bt, stopXPTicker)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	stopXPTicker <- true

	flushSession(bt)

	bt.Close()
	db.Close()
}
