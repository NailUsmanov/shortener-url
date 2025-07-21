// Package tasks используется для образования структуры фонового удаления URL.
package tasks

// DeleteTask представляет задачу на удаление URL по конкретному пользователю.
type DeleteTask struct {
	UserID    string
	ShortURLs []string
}
