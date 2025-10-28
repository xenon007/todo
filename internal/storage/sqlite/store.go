package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"todo/internal/models"
)

// Store wraps access to the SQLite database and exposes high level helpers.
type Store struct {
	db     *sql.DB
	logger *slog.Logger
}

// Open initializes a new SQLite store and runs the required migrations.
func Open(dbPath string, logger *slog.Logger) (*Store, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("empty database path")
	}

	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}

	if err := ensureDir(dbPath); err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_busy_timeout=5000&_foreign_keys=ON", dbPath))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	conn.SetMaxOpenConns(1)
	conn.SetConnMaxLifetime(0)

	s := &Store{db: conn, logger: logger}
	if err := s.migrate(); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return s, nil
}

// Close releases the database resources.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func ensureDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS projects (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL UNIQUE,
            color TEXT NOT NULL DEFAULT '#2563eb',
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        );`,
		`CREATE TABLE IF NOT EXISTS tasks (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            project_id INTEGER NOT NULL,
            title TEXT NOT NULL,
            description TEXT NOT NULL DEFAULT '',
            status TEXT NOT NULL DEFAULT 'todo',
            position INTEGER NOT NULL DEFAULT 0,
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
        );`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_project_status ON tasks(project_id, status);`,
		`CREATE TRIGGER IF NOT EXISTS trg_projects_updated
            AFTER UPDATE ON projects
            FOR EACH ROW BEGIN
                UPDATE projects SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
            END;`,
		`CREATE TRIGGER IF NOT EXISTS trg_tasks_updated
            AFTER UPDATE ON tasks
            FOR EACH ROW BEGIN
                UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
            END;`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

// ListProjects retrieves all projects ordered by creation date.
func (s *Store) ListProjects(ctx context.Context) ([]models.Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, color, created_at, updated_at FROM projects ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Color, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// CreateProject persists a new project with optional color.
func (s *Store) CreateProject(ctx context.Context, name, color string) (models.Project, error) {
	if strings.TrimSpace(name) == "" {
		return models.Project{}, fmt.Errorf("project name must not be empty")
	}
	if color == "" {
		color = randomPaletteColor()
	}

	res, err := s.db.ExecContext(ctx, `INSERT INTO projects(name, color) VALUES(?, ?)`, strings.TrimSpace(name), color)
	if err != nil {
		return models.Project{}, fmt.Errorf("insert project: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.Project{}, fmt.Errorf("project id: %w", err)
	}
	return s.GetProject(ctx, id)
}

// GetProject fetches a single project by id.
func (s *Store) GetProject(ctx context.Context, id int64) (models.Project, error) {
	var p models.Project
	err := s.db.QueryRowContext(ctx, `SELECT id, name, color, created_at, updated_at FROM projects WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Color, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Project{}, fmt.Errorf("project not found")
	}
	if err != nil {
		return models.Project{}, fmt.Errorf("get project: %w", err)
	}
	return p, nil
}

// UpdateProject renames a project and optionally changes its color.
func (s *Store) UpdateProject(ctx context.Context, id int64, name, color string) (models.Project, error) {
	if strings.TrimSpace(name) == "" {
		return models.Project{}, fmt.Errorf("project name must not be empty")
	}
	if color == "" {
		color = randomPaletteColor()
	}

	res, err := s.db.ExecContext(ctx, `UPDATE projects SET name = ?, color = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, strings.TrimSpace(name), color, id)
	if err != nil {
		return models.Project{}, fmt.Errorf("update project: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return models.Project{}, err
	}
	if affected == 0 {
		return models.Project{}, fmt.Errorf("project not found")
	}
	return s.GetProject(ctx, id)
}

// DeleteProject removes a project along with its tasks.
func (s *Store) DeleteProject(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("project not found")
	}
	return nil
}

// ListTasks returns tasks for the given project ordered by status and position.
func (s *Store) ListTasks(ctx context.Context, projectID int64) ([]models.Task, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, project_id, title, description, status, position, created_at, updated_at
        FROM tasks WHERE project_id = ? ORDER BY status, position, id`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Position, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// CreateTask inserts a new task for a project.
func (s *Store) CreateTask(ctx context.Context, t models.Task) (models.Task, error) {
	if strings.TrimSpace(t.Title) == "" {
		return models.Task{}, fmt.Errorf("task title must not be empty")
	}
	if _, ok := models.ValidTaskStatuses[t.Status]; !ok {
		t.Status = "todo"
	}

	pos, err := s.nextPosition(ctx, t.ProjectID, t.Status)
	if err != nil {
		return models.Task{}, err
	}

	res, err := s.db.ExecContext(ctx, `INSERT INTO tasks(project_id, title, description, status, position) VALUES(?, ?, ?, ?, ?)`, t.ProjectID, strings.TrimSpace(t.Title), strings.TrimSpace(t.Description), t.Status, pos)
	if err != nil {
		return models.Task{}, fmt.Errorf("insert task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.Task{}, fmt.Errorf("task id: %w", err)
	}
	return s.GetTask(ctx, id)
}

// GetTask retrieves a task by id.
func (s *Store) GetTask(ctx context.Context, id int64) (models.Task, error) {
	var t models.Task
	err := s.db.QueryRowContext(ctx, `SELECT id, project_id, title, description, status, position, created_at, updated_at FROM tasks WHERE id = ?`, id).
		Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Position, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Task{}, fmt.Errorf("task not found")
	}
	if err != nil {
		return models.Task{}, fmt.Errorf("get task: %w", err)
	}
	return t, nil
}

// UpdateTask updates task fields and moves the task between columns when needed.
func (s *Store) UpdateTask(ctx context.Context, id int64, changes map[string]any) (models.Task, error) {
	current, err := s.GetTask(ctx, id)
	if err != nil {
		return models.Task{}, err
	}

	title := current.Title
	description := current.Description
	status := current.Status
	position := current.Position

	if v, ok := changes["title"].(string); ok && strings.TrimSpace(v) != "" {
		title = strings.TrimSpace(v)
	}
	if v, ok := changes["description"].(string); ok {
		description = strings.TrimSpace(v)
	}
	if v, ok := changes["status"].(string); ok {
		if _, valid := models.ValidTaskStatuses[v]; valid {
			status = v
		}
	}

	if status != current.Status {
		pos, err := s.nextPosition(ctx, current.ProjectID, status)
		if err != nil {
			return models.Task{}, err
		}
		position = pos
	}

	_, err = s.db.ExecContext(ctx, `UPDATE tasks SET title = ?, description = ?, status = ?, position = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, title, description, status, position, id)
	if err != nil {
		return models.Task{}, fmt.Errorf("update task: %w", err)
	}
	return s.GetTask(ctx, id)
}

// DeleteTask removes a task by id.
func (s *Store) DeleteTask(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

func (s *Store) nextPosition(ctx context.Context, projectID int64, status string) (int64, error) {
	var position sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT MAX(position) FROM tasks WHERE project_id = ? AND status = ?`, projectID, status).Scan(&position)
	if err != nil {
		return 0, fmt.Errorf("select position: %w", err)
	}
	if position.Valid {
		return position.Int64 + 1, nil
	}
	return 0, nil
}

func randomPaletteColor() string {
	palette := []string{
		"#2563eb", // blue-600
		"#7c3aed", // violet-600
		"#dc2626", // red-600
		"#059669", // green-600
		"#ea580c", // orange-600
		"#d97706", // amber-600
		"#0ea5e9", // sky-500
	}
	rand.Seed(time.Now().UnixNano())
	return palette[rand.Intn(len(palette))]
}
