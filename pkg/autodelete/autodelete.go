package autodelete

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
	"go.uber.org/zap"
)

const (
	dbFileName = "./autodelete.sqlite"
	maxRetries = 3
)

// Manager manages autodelete tasks.
type Manager struct {
	// Bot client to delete messages with.
	Bot *gotgbot.Bot
	// Database which stores messages.
	DB *sqlx.DB
	// Mutex for safe access if needed
	mu sync.Mutex
	// Logger for logging internal operations
	Log *zap.Logger
}

// NewManager creates a new Manager from given bot.
func NewManager(bot *gotgbot.Bot, log *zap.Logger) (*Manager, error) {
	db, err := sqlx.Open("sqlite", dbFileName)
	if err != nil {
		log.Error("[autodelete][NewManager] failed to open database", zap.Error(err))
		return nil, err
	}

	// Set connection limits for stability
	db.SetMaxOpenConns(1)

	createTableSQL := `CREATE TABLE IF NOT EXISTS autodelete (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        chat_id INTEGER,
        message_id INTEGER,
        expiry_time DATETIME,
        UNIQUE(chat_id, message_id)
    );`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Error("[autodelete][NewManager] failed to create table", zap.Error(err))
		return nil, err
	}

	// Cleanup any corrupted entries with 0 IDs
	_, _ = db.Exec("DELETE FROM autodelete WHERE chat_id = 0 OR message_id = 0")

	return &Manager{
		Bot: bot,
		DB:  db,
		Log: log,
	}, nil
}

// SaveMessage adds a message to the autodelete database which will be deleted after duration.
func (m *Manager) SaveMessage(msg *gotgbot.Message, duration time.Duration) error {
	if msg == nil || msg.Chat.Id == 0 || msg.MessageId == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	data := MessageData{
		ChatId:     msg.Chat.Id,
		MessageId:  msg.MessageId,
		ExpiryTime: time.Now().Add(duration),
	}

	const insertQuery = `INSERT INTO autodelete (chat_id, message_id, expiry_time) VALUES (:chat_id, :message_id, :expiry_time)
		ON CONFLICT(chat_id, message_id) DO UPDATE SET expiry_time=excluded.expiry_time;`

	_, err := m.DB.NamedExec(insertQuery, data)
	if err != nil {
		m.Log.Error("[autodelete][SaveMessage] failed to save message",
			zap.Int64("chat_id", data.ChatId),
			zap.Int64("message_id", data.MessageId),
			zap.Error(err))
	}

	return err
}

// SaveData adds a message to the autodelete database using explicit IDs.
func (m *Manager) SaveData(chatId, messageId int64, duration time.Duration) error {
	if chatId == 0 || messageId == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	data := MessageData{
		ChatId:     chatId,
		MessageId:  messageId,
		ExpiryTime: time.Now().Add(duration),
	}

	const insertQuery = `INSERT INTO autodelete (chat_id, message_id, expiry_time) VALUES (:chat_id, :message_id, :expiry_time)
		ON CONFLICT(chat_id, message_id) DO UPDATE SET expiry_time=excluded.expiry_time;`

	_, err := m.DB.NamedExec(insertQuery, data)
	if err != nil {
		m.Log.Error("[autodelete][SaveData] failed to save data",
			zap.Int64("chat_id", data.ChatId),
			zap.Int64("message_id", data.MessageId),
			zap.Error(err))
	}

	return err
}

// Remove removes a message from the database.
func (m *Manager) Remove(chatId, messageId int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	const deleteQuery = `DELETE FROM autodelete WHERE chat_id = ? AND message_id = ?`
	_, err := m.DB.Exec(deleteQuery, chatId, messageId)
	if err != nil {
		m.Log.Warn("[autodelete][Remove] failed to delete entry", zap.Error(err))
	}
	return err
}

// Run starts the autodelete system.
func (m *Manager) Run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 1024)
			n := runtime.Stack(buf, false)
			m.Log.Error("[autodelete][Run] recovered from panic",
				zap.Any("panic", r),
				zap.String("stack", string(buf[:n])))
			// Restart the worker
			go m.Run(ctx)
		}
	}()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.processExpired()
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) processExpired() {
	const selectQuery = `SELECT chat_id, message_id, expiry_time FROM autodelete WHERE expiry_time <= ?`
	var result []MessageData

	m.mu.Lock()
	err := m.DB.Select(&result, selectQuery, time.Now())
	m.mu.Unlock()

	if err != nil {
		m.Log.Error("[autodelete][processExpired] select query failed", zap.Error(err))
		return
	}

	for _, r := range result {
		go func(data MessageData) {
			defer func() {
				if r := recover(); r != nil {
					m.Log.Error("[autodelete][deleteTask] panic recovered", zap.Any("panic", r))
				}
			}()

			var success bool
			for i := 0; i < maxRetries; i++ {
				_, err := m.Bot.DeleteMessage(data.ChatId, data.MessageId, nil)
				if err == nil {
					success = true
					break
				}
				// If message already deleted or forbidden, stop retrying
				if strings.Contains(err.Error(), "message to delete not found") ||
					strings.Contains(err.Error(), "message can't be deleted") {
					success = true // Treat as handled
					break
				}

				m.Log.Info("[autodelete][deleteTask] retry",
					zap.Int64("chat_id", data.ChatId),
					zap.Int64("message_id", data.MessageId),
					zap.Int("attempt", i+1),
					zap.Error(err))
				time.Sleep(time.Second * time.Duration(i+1))
			}

			if success {
				_ = m.Remove(data.ChatId, data.MessageId)
			}
		}(r)
	}
}

// MessageData holds data about message to delete.
type MessageData struct {
	ChatId     int64     `db:"chat_id"`
	MessageId  int64     `db:"message_id"`
	ExpiryTime time.Time `db:"expiry_time"`
}
