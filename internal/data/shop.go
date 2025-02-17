package data

import (
	"database/sql"
	"errors"
	"strings"
)

type User struct {
	ID       int64  `json:"id"`
	Balance  int    `json:"coins"`
	Username string `json:"username"`
	Password string `json:"-"`
}
type Item struct {
	ID       int64  `json:"-"`
	Name     string `json:"item"`
	Price    int    `json:"-"`
	Quantity int    `json:"quantity"`
}
type UserItem struct {
	User_id  int64
	Item_id  int64
	Quantity int
}
type Transcation struct {
	ID         int64
	FromUserID int
	ToUserID   int
	Amount     int
}

type ShopModel struct {
	DB *sql.DB
}

func (m *ShopModel) GetUserByUsername(username string) (*User, error) {
	stmt := `SELECT id, username, password, balance FROM users WHERE username = $1`

	row := m.DB.QueryRow(stmt, username)

	var user User
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Balance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (m *ShopModel) InsertUser(username string, password string) (*User, error) {
	stmt := `INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id, balance`

	var newUser User
	err := m.DB.QueryRow(stmt, username, password).Scan(&newUser.ID, &newUser.Balance)
	if err != nil {
		return nil, err
	}
	newUser.Username = username
	return &newUser, nil
}

func (m *ShopModel) GetUserBalanceAndInventory(userID int64) (int, []Item, error) {
	stmt := `
        SELECT u.balance, i.id, i.name, i.price, ui.quantity
        FROM users u
        LEFT JOIN user_items ui ON u.id = ui.user_id
        LEFT JOIN items i ON ui.item_id = i.id
        WHERE u.id = $1
    `

	rows, err := m.DB.Query(stmt, userID)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var balance int
	inventoryItems := []Item{}

	hasRows := false
	for rows.Next() {

		var currentBalance int
		var itemID sql.NullInt64
		var name sql.NullString
		var price sql.NullInt64
		var quantity sql.NullInt64

		if err := rows.Scan(&currentBalance, &itemID, &name, &price, &quantity); err != nil {
			return 0, nil, err
		}

		// Set balance from the first row
		if !hasRows {
			balance = currentBalance
		}

		hasRows = true

		// Handle NULL values for name, price, and quantity
		item := Item{
			ID:       itemID.Int64,        // If itemID is NULL, this will be 0
			Name:     name.String,         // If name is NULL, this will be an empty string
			Price:    int(price.Int64),    // If price is NULL, this will be 0
			Quantity: int(quantity.Int64), // If quantity is NULL, this will be 0
		}

		// Only append the item if it has a valid name (i.e., not NULL)
		if name.Valid {
			inventoryItems = append(inventoryItems, item)
		}
	}

	if err := rows.Err(); err != nil {
		return 0, nil, err
	}

	// If no rows were returned, fetch the user's balance separately
	if !hasRows {
		balance, err = m.GetUserBalance(userID)
		if err != nil {
			return 0, nil, err
		}
	}

	return balance, inventoryItems, nil
}

func (m *ShopModel) GetUserBalance(userID int64) (int, error) {
	stmt := `SELECT balance FROM users WHERE id = $1`
	var balance int
	err := m.DB.QueryRow(stmt, userID).Scan(&balance)
	if err != nil {
		return 0, err
	}
	return balance, nil
}

func (m *ShopModel) UpdateSenderBalance(tx *sql.Tx, userID int64, amount int) error {
	stmt := `UPDATE users SET balance = balance - $1 WHERE id = $2`
	_, err := tx.Exec(stmt, amount, userID)
	return err
}

func (m *ShopModel) UpdateReceiverBalance(tx *sql.Tx, userID int64, amount int) error {
	stmt := `UPDATE users SET balance = balance + $1 WHERE id = $2`
	_, err := tx.Exec(stmt, amount, userID)
	return err
}

func (m *ShopModel) InsertTransaction(tx *sql.Tx, fromUserID int64, toUserID int64, amount int) error {
	stmt := `
		INSERT INTO transactions (from_user_id, to_user_id, amount)
		VALUES ($1, $2, $3)
	`
	_, err := tx.Exec(stmt, fromUserID, toUserID, amount)
	return err
}

func (m *ShopModel) GetItemPrice(itemName string) (int, error) {
	stmt := `SELECT price FROM items WHERE name = $1`
	var price int
	err := m.DB.QueryRow(stmt, itemName).Scan(&price)
	if err != nil {
		return 0, err
	}
	return price, nil
}

func (m *ShopModel) UpdateUserBalanceAfterPurchase(tx *sql.Tx, userID int64, itemPrice int) error {
	stmt := `UPDATE users SET balance = balance - $1 WHERE id = $2`
	_, err := tx.Exec(stmt, itemPrice, userID)
	return err
}

func (m *ShopModel) InsertUserItem(tx *sql.Tx, userID int64, itemID int64) error {
	stmt := `INSERT INTO user_items (user_id, item_id) VALUES ($1, $2)
	ON CONFLICT (user_id, item_id)
	DO UPDATE SET quantity = user_items.quantity+1
	`
	_, err := tx.Exec(stmt, userID, itemID)
	return err
}

func (m *ShopModel) GetAllItems() ([]Item, error) {
	stmt := `SELECT id, name, price FROM items`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item

	for rows.Next() {
		var i Item
		err := rows.Scan(&i.ID, &i.Name, &i.Price)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (m *ShopModel) CheckUserOwnItem(userID int64, itemID int64) (bool, error) {
	stmt := `SELECT EXISTS(SELECT 1 FROM user_items WHERE user_id = $1 AND item_id = $2)`
	var exists bool
	err := m.DB.QueryRow(stmt, userID, itemID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (m *ShopModel) GetUserByID(userID int64) (*User, error) {
	stmt := `SELECT id, username, balance FROM users WHERE id = $1`

	row := m.DB.QueryRow(stmt, userID)

	var user User
	err := row.Scan(&user.ID, &user.Username, &user.Balance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (m *ShopModel) GetItemByName(itemName string) (*Item, error) {
	stmt := `SELECT id, name, price FROM items WHERE name = $1`

	row := m.DB.QueryRow(stmt, itemName)

	var item Item
	err := row.Scan(&item.ID, &item.Name, &item.Price)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (m *ShopModel) GetItemByID(itemID int64) (*Item, error) {
	stmt := `SELECT id, name, price FROM items WHERE id = $1`

	row := m.DB.QueryRow(stmt, itemID)

	var item Item
	err := row.Scan(&item.ID, &item.Name, &item.Price)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (m *ShopModel) GetUsersByIDs(userIDs []int64) (map[int64]*User, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	query := `SELECT id, username FROM users WHERE id IN (` + strings.Repeat("?,", len(userIDs)-1) + `?)`
	args := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		args[i] = id
	}

	rows, err := m.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make(map[int64]*User)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username); err != nil {
			return nil, err
		}
		users[user.ID] = &user
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (m *ShopModel) GetTransactionHistoryWithUsernames(userID int64) ([]struct {
	FromUserID int64
	ToUserID   int64
	FromUser   string
	ToUser     string
	Amount     int
}, error) {
	stmt := `
		SELECT
			t.from_user_id, t.to_user_id,
			u1.username AS from_user,
			u2.username AS to_user,
			t.amount
		FROM transactions t
		LEFT JOIN users u1 ON t.from_user_id = u1.id
		LEFT JOIN users u2 ON t.to_user_id = u2.id
		WHERE t.from_user_id = $1 OR t.to_user_id = $1
		ORDER BY t.id DESC
	`

	rows, err := m.DB.Query(stmt, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []struct {
		FromUserID int64
		ToUserID   int64
		FromUser   string
		ToUser     string
		Amount     int
	}

	for rows.Next() {
		var t struct {
			FromUserID int64
			ToUserID   int64
			FromUser   sql.NullString
			ToUser     sql.NullString
			Amount     int
		}
		if err := rows.Scan(&t.FromUserID, &t.ToUserID, &t.FromUser, &t.ToUser, &t.Amount); err != nil {
			return nil, err
		}

		// Convert sql.NullString to string, handling NULL values
		fromUser := ""
		if t.FromUser.Valid {
			fromUser = t.FromUser.String
		}

		toUser := ""
		if t.ToUser.Valid {
			toUser = t.ToUser.String
		}

		transactions = append(transactions, struct {
			FromUserID int64
			ToUserID   int64
			FromUser   string
			ToUser     string
			Amount     int
		}{
			FromUserID: t.FromUserID,
			ToUserID:   t.ToUserID,
			FromUser:   fromUser,
			ToUser:     toUser,
			Amount:     t.Amount,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}

/*
func (m *ShopModel) GetTransactionHistoryWithUsernames(userID int64) ([]struct {
	FromUserID int64
	ToUserID   int64
	FromUser   string
	ToUser     string
	Amount     int
}, error) {
	stmt := `
		SELECT
			t.from_user_id, t.to_user_id,
			u1.username AS from_user,
			u2.username AS to_user,
			t.amount
		FROM transactions t
		LEFT JOIN users u1 ON t.from_user_id = u1.id
		LEFT JOIN users u2 ON t.to_user_id = u2.id
		WHERE t.from_user_id = $1 OR t.to_user_id = $1
		ORDER BY t.id DESC
	`

	rows, err := m.DB.Query(stmt, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []struct {
		FromUserID int64
		ToUserID   int64
		FromUser   string
		ToUser     string
		Amount     int
	}

	for rows.Next() {
		var t struct {
			FromUserID int64
			ToUserID   int64
			FromUser   string
			ToUser     string
			Amount     int
		}
		if err := rows.Scan(&t.FromUserID, &t.ToUserID, &t.FromUser, &t.ToUser, &t.Amount); err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}
*/
