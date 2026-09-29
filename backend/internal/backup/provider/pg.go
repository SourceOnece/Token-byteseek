package provider

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/backup"
	postgresinfra "github.com/TokenFlux/TokenRouter/internal/infra/postgres"
)

// PgDumper implements backup.DBDumper using pg_dump/psql
// DatabaseOptions 是装配传入的数据库连接快照。
type DatabaseOptions struct {
	Host                            string
	Port                            int
	User, Password, DBName, SSLMode string
}
type PgDumper struct {
	cfg            *DatabaseOptions
	db             *sql.DB
	commandContext func(context.Context, string, ...string) *exec.Cmd
}

// NewPgDumper creates a new PgDumper
func NewPgDumper(cfg DatabaseOptions, database ...*sql.DB) backup.DBDumper {
	var db *sql.DB
	if len(database) > 0 {
		db = database[0]
	}
	return &PgDumper{cfg: &cfg, db: db}
}

// Dump executes pg_dump and returns a streaming reader of the output
func (d *PgDumper) Dump(ctx context.Context, opts backup.BackupDumpOptions) (io.ReadCloser, error) {
	if d.db == nil {
		return nil, errors.New("acquire backup migration lock: nil sql db")
	}
	lockConn, err := d.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if err := postgresinfra.AcquireMigrationLock(ctx, lockConn); err != nil {
		discardSQLConnection(lockConn)
		return nil, err
	}
	releaseLock := func() error { return releaseBackupMigrationLock(lockConn) }
	args := []string{
		"-h", d.cfg.Host,
		"-p", fmt.Sprintf("%d", d.cfg.Port),
		"-U", d.cfg.User,
		"-d", d.cfg.DBName,
		"--no-owner",
		"--no-acl",
		"--clean",
		"--if-exists",
	}
	for _, tablePattern := range opts.ExcludeTableData {
		// 只跳过表数据，保留结构、约束和索引，避免恢复后缺表。
		args = append(args, "--exclude-table-data="+tablePattern)
	}

	commandContext := d.commandContext
	if commandContext == nil {
		commandContext = exec.CommandContext
	}
	cmd := commandContext(ctx, "pg_dump", args...)
	if d.cfg.Password != "" {
		cmd.Env = append(cmd.Environ(), "PGPASSWORD="+d.cfg.Password)
	}
	if d.cfg.SSLMode != "" {
		cmd.Env = append(cmd.Environ(), "PGSSLMODE="+d.cfg.SSLMode)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("create stdout pipe: %w", err), releaseLock())
	}

	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		return nil, errors.Join(fmt.Errorf("start pg_dump: %w", err), releaseLock())
	}

	return &cmdReadCloser{ReadCloser: stdout, cmd: cmd, release: releaseLock}, nil
}

func releaseBackupMigrationLock(conn *sql.Conn) error {
	unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := postgresinfra.ReleaseMigrationLock(unlockCtx, conn); err != nil {
		// 解锁失败时不能把可能仍持锁的会话交还连接池。
		discardSQLConnection(conn)
		return fmt.Errorf("release backup migration lock: %w", err)
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("close backup migration lock connection: %w", err)
	}
	return nil
}

func discardSQLConnection(conn *sql.Conn) {
	_ = conn.Raw(func(any) error { return driver.ErrBadConn })
	_ = conn.Close()
}

// Restore executes psql to restore from a streaming reader
// @project-doc docs/operations/deployment_and_migrations.md#maintenance_execution
func (d *PgDumper) Restore(ctx context.Context, data io.Reader) error {
	args := []string{
		"-h", d.cfg.Host,
		"-p", fmt.Sprintf("%d", d.cfg.Port),
		"-U", d.cfg.User,
		"-d", d.cfg.DBName,
		"--single-transaction",
		"--set=ON_ERROR_STOP=1",
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, "psql", args...)
	if d.cfg.Password != "" {
		cmd.Env = append(cmd.Environ(), "PGPASSWORD="+d.cfg.Password)
	}
	if d.cfg.SSLMode != "" {
		cmd.Env = append(cmd.Environ(), "PGSSLMODE="+d.cfg.SSLMode)
	}

	// 输入只有完整读取后才关闭管道，损坏归档先取消 psql，避免 EOF 触发提交。
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}
	_, copyErr := io.Copy(stdin, data)
	if copyErr != nil {
		cancel()
	}
	closeErr := stdin.Close()
	waitErr := cmd.Wait()
	if err := errors.Join(copyErr, closeErr, waitErr); err != nil {
		return fmt.Errorf("%w: %s", err, output.String())
	}
	return nil
}

// cmdReadCloser wraps a command stdout pipe and waits for the process on Close
type cmdReadCloser struct {
	io.ReadCloser
	cmd      *exec.Cmd
	once     sync.Once
	closeErr error
	release  func() error
}

func (c *cmdReadCloser) Close() error {
	c.once.Do(func() {
		_ = c.ReadCloser.Close()
		if err := c.cmd.Wait(); err != nil {
			c.closeErr = fmt.Errorf("pg_dump exited with error: %w", err)
		}
		if c.release != nil {
			c.closeErr = errors.Join(c.closeErr, c.release())
		}
	})
	return c.closeErr
}
