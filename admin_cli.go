package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/joho/godotenv"
)

func runAdminCLI(args []string, input *os.File, output io.Writer) error {
	if len(args) == 0 || args[0] != "promote-root" {
		return errors.New("usage: new-api admin promote-root --user-id <ID>")
	}
	flags := flag.NewFlagSet("promote-root", flag.ContinueOnError)
	flags.SetOutput(output)
	userID := flags.Int("user-id", 0, "existing user ID to promote")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if *userID <= 0 || flags.NArg() != 0 {
		return errors.New("usage: new-api admin promote-root --user-id <ID>")
	}
	stat, err := input.Stat()
	if err != nil || stat.Mode()&os.ModeCharDevice == 0 {
		return errors.New("promotion requires an interactive terminal")
	}
	_ = godotenv.Load(".env")
	if os.Getenv("REDIS_CONN_STRING") != "" && os.Getenv("SESSION_SECRET") == "" {
		return errors.New("SESSION_SECRET must match the running service when Redis is enabled")
	}
	if dsn := os.Getenv("SQL_DSN"); dsn == "" || strings.HasPrefix(dsn, "local") {
		sqlitePath := os.Getenv("SQLITE_PATH")
		if sqlitePath == "" {
			sqlitePath = "one-api.db"
		}
		sqlitePath = strings.SplitN(sqlitePath, "?", 2)[0]
		if _, err := os.Stat(sqlitePath); err != nil {
			return fmt.Errorf("configured SQLite database %q is not accessible: %w", sqlitePath, err)
		}
	}
	// InitEnv parses server flags. The admin command has already parsed its own
	// arguments, so only the executable name is passed to the server parser.
	originalArgs := os.Args
	os.Args = os.Args[:1]
	defer func() { os.Args = originalArgs }()
	common.InitEnv()
	if err := model.InitDBForMaintenance(); err != nil {
		return err
	}
	defer model.CloseDB()
	if err := model.InitLogDBForMaintenance(); err != nil {
		return err
	}
	logDB, err := model.LOG_DB.DB()
	if err != nil {
		return err
	}
	if err := logDB.Ping(); err != nil {
		return fmt.Errorf("audit database is unavailable: %w", err)
	}
	if !model.LOG_DB.Migrator().HasTable(&model.Log{}) {
		return errors.New("audit log table is missing; refusing to promote without audit storage")
	}
	if err := common.InitRedisClient(); err != nil {
		return err
	}
	user, err := model.GetUserById(*userID, false)
	if err != nil {
		return fmt.Errorf("target user not found: %w", err)
	}
	if user.Role == common.RoleRootUser {
		return model.ErrRootAlreadyPromoted
	}
	if user.Status != common.UserStatusEnabled ||
		(user.Role != common.RoleCommonUser && user.Role != common.RoleAdminUser) {
		return model.ErrRootPromotionIneligible
	}
	if err := confirmRootPromotion(input, output, user); err != nil {
		return err
	}
	promotionErr := model.PromoteUserToRoot(*userID)
	if promotionErr != nil {
		if errors.Is(promotionErr, model.ErrRootAlreadyPromoted) {
			return promotionErr
		}
		// Cache publication or session revocation can fail after the role
		// transaction commits. Audit that committed change even on a partial error.
		current, lookupErr := model.GetUserById(*userID, false)
		if lookupErr != nil || current.Role != common.RoleRootUser {
			return promotionErr
		}
	}
	hostname, _ := os.Hostname()
	if err := model.RecordOperationAuditLog(user.Id,
		fmt.Sprintf("Local CLI promoted user %s to super administrator", user.Username),
		"local-cli", "user.promote_root.cli",
		map[string]interface{}{"target_user_id": user.Id, "username": user.Username},
		map[string]interface{}{"source": "local_cli", "hostname": hostname}, nil); err != nil {
		return errors.Join(promotionErr, fmt.Errorf("role was promoted but audit logging failed: %w", err))
	}
	if promotionErr != nil {
		return promotionErr
	}
	fmt.Fprintf(output, "User %s (ID %d) is now a super administrator. Existing sessions were revoked.\n", user.Username, user.Id)
	return nil
}

func confirmRootPromotion(input io.Reader, output io.Writer, user *model.User) error {
	expected := "PROMOTE " + strconv.Itoa(user.Id)
	fmt.Fprintf(output, "Promote user %s (ID %d, role %d) to super administrator?\n", user.Username, user.Id, user.Role)
	fmt.Fprintf(output, "Type %s to confirm: ", expected)
	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if strings.TrimSpace(line) != expected {
		return errors.New("promotion cancelled")
	}
	return nil
}
