package dbaccess

import (
	"MessagePushService/interfaces"
	"context"
	"database/sql"
	"log"
	"strings"
	"sync"
)

var (
	dbMessagePersistenceOnce     sync.Once
	dbMessagePersistenceInstance *dbMessagePersistence
)

type dbMessagePersistence struct {
	db *sql.DB
}

func NewDBMessagePersistence(db *sql.DB) interfaces.DBMessagePersistence {
	dbMessagePersistenceOnce.Do(func() {
		dbMessagePersistenceInstance = &dbMessagePersistence{db: db}
	})

	return dbMessagePersistenceInstance
}

func (dbMessagePersistence *dbMessagePersistence) Add(ctx context.Context, userIDs []string, message *interfaces.DBMessage) (err error) {
	tx, err := dbMessagePersistence.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			log.Printf("[ERROR] add message error: %v", err)
			tx.Rollback()
			return
		}

		cErr := tx.Commit()
		if cErr != nil {
			log.Printf("[ERROR] commit tx error: %v", cErr)
		}
	}()

	strSQL1 := `
	INSERT INTO t_message 
		(message_id, message_type, content) 
	VALUES (?, ?, ?)
`
	strSQL2 := `
	INSERT INTO t_user_message 
		(user_id, message_id, status) 
	VALUES 
`
	var placeholders []string = make([]string, 0, len(userIDs))
	var args []interface{} = make([]interface{}, 0, len(userIDs))
	for _, userID := range userIDs {
		placeholders = append(placeholders, "(?, ?, ?)")
		args = append(args, userID, message.MessageID, interfaces.MessageStatusUnsent)
	}
	strSQL2 += strings.Join(placeholders, ",")

	_, err = tx.ExecContext(ctx, strSQL1, message.MessageID, message.MessageType, message.Content)
	if err != nil {
		log.Printf("[ERROR] exec strSQL1 error: %v", err)
		return err
	}
	log.Printf("[DEBUG] strSQL2: %s, args: %v", strSQL2, args)
	_, err = tx.ExecContext(ctx, strSQL2, args...)
	if err != nil {
		log.Printf("[ERROR] exec strSQL2 error: %v", err)
		return err
	}

	return nil
}

func (dbMessagePersistence *dbMessagePersistence) CheckExists(ctx context.Context, messageID string) (exists bool, err error) {
	err = dbMessagePersistence.db.QueryRowContext(ctx, "SELECT 1 FROM t_message WHERE message_id = ? limit 1", messageID).Scan(&exists)
	log.Println(err)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (dbMessagePersistence *dbMessagePersistence) GetPendingMessage(ctx context.Context) (out *interfaces.DBMessage, userIDs []string, err error) {
	out = &interfaces.DBMessage{}
	err = dbMessagePersistence.db.
		QueryRowContext(ctx, "SELECT * FROM t_message order by created_at asc limit 1").
		Scan(&out.MessageID, &out.MessageType, &out.Content, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	rows, err := dbMessagePersistence.db.QueryContext(ctx, "SELECT user_id FROM t_user_message WHERE message_id = ?", out.MessageID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var userID string
		err = rows.Scan(&userID)
		if err != nil {
			return nil, nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return out, userIDs, nil
}

func (dbMessagePersistence *dbMessagePersistence) GetPendingMessagesByUserID(ctx context.Context, userID string, limit int) (out []*interfaces.DBMessage, err error) {
	strSQL := `
		SELECT * FROM t_message WHERE message_id IN (
			SELECT message_id FROM t_user_message WHERE user_id = ? AND status = ? ORDER BY created_at ASC LIMIT ?
		)
	`
	rows, err := dbMessagePersistence.db.QueryContext(ctx, strSQL, userID, interfaces.MessageStatusUnsent, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		tmp := &interfaces.DBMessage{}
		err = rows.Scan(&tmp.MessageID, &tmp.MessageType, &tmp.Content, &tmp.CreatedAt, &tmp.UpdatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, tmp)
	}

	return
}

func (dbMessagePersistence *dbMessagePersistence) UpdateStatus(ctx context.Context, userID, msgID string, status int) error {
	strSQL := `
		UPDATE t_user_message
		SET status = ?
		WHERE user_id = ? AND message_id = ?
	`
	_, err := dbMessagePersistence.db.ExecContext(ctx, strSQL, status, userID, msgID)
	if err != nil {
		return err
	}
	return nil
}
