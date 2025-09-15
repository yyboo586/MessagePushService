package dbaccess

import (
	"MessagePushService/interfaces"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
)

var (
	dbMessageOnce     sync.Once
	dbMessageInstance *dbMessage
)

type dbMessage struct {
	db *sql.DB
}

func NewDBMessage(db *sql.DB) interfaces.IDBMessage {
	dbMessageOnce.Do(func() {
		dbMessageInstance = &dbMessage{db: db}
	})

	return dbMessageInstance
}

func (m *dbMessage) Add(ctx context.Context, userIDs []string, message *interfaces.DBMessage) (err error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}

		cErr := tx.Commit()
		if cErr != nil {
			err = cErr
			return
		}
	}()

	strSQL1 := `
	INSERT INTO t_message 
		(id, type, content, timestamp) 
	VALUES (?, ?, ?, ?)
`
	strSQL2 := `
	INSERT INTO t_user_message 
		(user_id, message_id) 
	VALUES 
`
	var placeholders []string = make([]string, 0, len(userIDs))
	var args []interface{} = make([]interface{}, 0, len(userIDs))
	for _, userID := range userIDs {
		placeholders = append(placeholders, "(?, ?)")
		args = append(args, userID, message.ID)
	}
	strSQL2 += strings.Join(placeholders, ",")

	_, err = tx.ExecContext(ctx, strSQL1, message.ID, message.Type, message.Content, message.Timestamp)
	if err != nil {
		return
	}
	_, err = tx.ExecContext(ctx, strSQL2, args...)
	if err != nil {
		return
	}

	return
}

func (m *dbMessage) GetByID(ctx context.Context, messageID string) (out *interfaces.DBMessage, userIDs []string, err error) {
	out = &interfaces.DBMessage{}
	userIDs = make([]string, 0)
	strSQL := `
		SELECT 
			m.id, m.type, m.content, m.timestamp, m.created_at, m.updated_at
		FROM t_message m 
		WHERE 
			m.id = ?
	`
	err = m.db.
		QueryRowContext(ctx, strSQL, messageID).
		Scan(&out.ID, &out.Type, &out.Content, &out.Timestamp, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			err = fmt.Errorf("%w, messageID: %s", interfaces.ErrRecordNotFound, messageID)
			return
		}
		return
	}

	rows, err := m.db.QueryContext(ctx, "SELECT user_id FROM t_user_message WHERE message_id = ?", messageID)
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

	return
}

func (m *dbMessage) GetByPushStatus(ctx context.Context, status interfaces.MessagePushStatus) (out *interfaces.DBMessage, userIDs []string, err error) {
	out = &interfaces.DBMessage{}
	err = m.db.
		QueryRowContext(ctx, "SELECT * FROM t_message WHERE push_status = ? order by created_at asc limit 1", status).
		Scan(&out.ID, &out.Type, &out.Content, &out.Timestamp, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			err = fmt.Errorf("%w, status: %d", interfaces.ErrRecordNotFound, status)
			return
		}
		return
	}

	rows, err := m.db.QueryContext(ctx, "SELECT user_id FROM t_user_message WHERE message_id = ?", out.ID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var userID string
		err = rows.Scan(&userID)
		if err != nil {
			return
		}
		userIDs = append(userIDs, userID)
	}

	return
}

func (m *dbMessage) GetByUserID(ctx context.Context, userID string, status interfaces.MessagePushStatus, limit int) (out []*interfaces.DBMessage, err error) {
	strSQL := `
		SELECT
			id,
			type,
			content,
			timestamp,
			created_at,
			updated_at
		FROM t_message WHERE id IN (
			SELECT message_id FROM t_user_message WHERE user_id = ? AND push_status = ? ORDER BY created_at ASC
		) LIMIT ?
	`
	rows, err := m.db.QueryContext(ctx, strSQL, userID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		tmp := &interfaces.DBMessage{}
		err = rows.Scan(&tmp.ID, &tmp.Type, &tmp.Content, &tmp.Timestamp, &tmp.CreatedAt, &tmp.UpdatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, tmp)
	}

	return
}

func (m *dbMessage) UpdateStatus(ctx context.Context, userID, msgID string, status interfaces.MessagePushStatus) (err error) {
	strSQL := `
		UPDATE t_user_message
		SET 
			push_status = ?
		WHERE 
			user_id = ? AND message_id = ?
	`
	_, err = m.db.ExecContext(ctx, strSQL, status, userID, msgID)
	if err != nil {
		return
	}
	return
}
