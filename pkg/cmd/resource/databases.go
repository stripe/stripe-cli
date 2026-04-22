package resource

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/cmdutil"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

const (
	databaseJSONFlagName               = "json"
	databaseRequestVersion             = "unsafe-development"
	databaseDeleteConfirmationPhrase   = "remove StripeDB"
	databaseUserDeleteConfirmationText = "remove user"
)

const databasesLongDescription = `Manage StripeDB.

These commands target unstable preview APIs and may change without notice.`

type databaseConnection struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	DatabaseName string `json:"database_name"`
}

type databaseUser struct {
	ID       string `json:"id"`
	Object   string `json:"object"`
	Livemode bool   `json:"livemode"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
	Created  string `json:"created"`
}

type databaseObject struct {
	ID         string             `json:"id"`
	Object     string             `json:"object"`
	Livemode   bool               `json:"livemode"`
	Created    string             `json:"created"`
	Status     string             `json:"status"`
	APIVersion string             `json:"api_version"`
	Connection databaseConnection `json:"connection"`
	User       *databaseUser      `json:"user"`
}

type databaseEnvelope struct {
	Database databaseObject `json:"database"`
}

type databaseUserEnvelope struct {
	DatabaseUser databaseUser `json:"database_user"`
}

type dataListEnvelope[T any] struct {
	Data []T `json:"data"`
}

// Swapped in tests so relative-time output stays deterministic.
var databaseNow = time.Now

type databaseDetailField struct {
	Label string
	Value string
}

var (
	databaseCreateOperationSpec = OperationSpec{
		Name:   "create",
		Path:   "/v2/data/databases",
		Method: http.MethodPost,
		Params: map[string]*ParamSpec{
			"api_version": {Type: "string"},
		},
	}
	databaseRetrieveOperationSpec = OperationSpec{
		Name:   "retrieve",
		Path:   "/v2/data/databases/{db_id}",
		Method: http.MethodGet,
	}
	databaseListOperationSpec = OperationSpec{
		Name:   "list",
		Path:   "/v2/data/databases",
		Method: http.MethodGet,
	}
	databaseDeleteOperationSpec = OperationSpec{
		Name:   "delete",
		Path:   "/v2/data/databases/{db_id}",
		Method: http.MethodDelete,
	}
	databaseUsersCreateOperationSpec = OperationSpec{
		Name:   "create",
		Path:   "/v2/data/databases/{db_id}/users",
		Method: http.MethodPost,
		Params: map[string]*ParamSpec{
			"username": {Type: "string"},
		},
	}
	databaseUsersRetrieveOperationSpec = OperationSpec{
		Name:   "retrieve",
		Path:   "/v2/data/databases/{db_id}/users/{dbuser_id}",
		Method: http.MethodGet,
	}
	databaseUsersListOperationSpec = OperationSpec{
		Name:   "list",
		Path:   "/v2/data/databases/{db_id}/users",
		Method: http.MethodGet,
	}
	databaseUsersDeleteOperationSpec = OperationSpec{
		Name:   "delete",
		Path:   "/v2/data/databases/{db_id}/users/{dbuser_id}",
		Method: http.MethodDelete,
	}
)

// AddDatabasesCmd registers the hand-written StripeDB command tree. If a
// generated command with the same name ever appears, prefer the manual version
// until the custom implementation is removed.
func AddDatabasesCmd(rootCmd *cobra.Command, cfg *config.Config) error {
	if rootCmd.Annotations == nil {
		rootCmd.Annotations = make(map[string]string)
	}

	if existing, ok := cmdutil.FindSubCmd(rootCmd, "databases"); ok {
		rootCmd.RemoveCommand(existing)
	}

	newDatabasesResourceCmd(rootCmd, cfg)
	return nil
}

func newDatabasesCmd(cfg *config.Config) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:         "stripe",
		Annotations: make(map[string]string),
	}
	databasesCmd := newDatabasesResourceCmd(rootCmd, cfg).Cmd
	rootCmd.RemoveCommand(databasesCmd)
	return databasesCmd
}

func newDatabasesResourceCmd(parentCmd *cobra.Command, cfg *config.Config) *ResourceCmd {
	databasesCmd := NewResourceCmd(parentCmd, "databases")
	databasesCmd.Cmd.Short = "Manage StripeDB (unstable preview APIs)"
	databasesCmd.Cmd.Long = databasesLongDescription
	databasesCmd.Cmd.PersistentFlags().Bool(databaseJSONFlagName, false, "Return JSON output instead of formatted text")
	addDatabaseCommands(databasesCmd.Cmd, cfg)
	databasesCmd.Cmd.Hidden = true

	if existing, ok := cmdutil.FindSubCmd(parentCmd, "databases"); ok {
		existing.Short = databasesCmd.Cmd.Short
		existing.Long = databasesCmd.Cmd.Long
		existing.Hidden = true
	}

	return databasesCmd
}

func addDatabaseCommands(root *cobra.Command, cfg *config.Config) {
	usersCmd := NewResourceCmd(root, "users")
	usersCmd.Cmd.Short = "Manage StripeDB users"
	usersCmd.Cmd.Long = databasesLongDescription

	newDatabaseCreateCmd(root, cfg)
	newDatabaseRetrieveCmd(root, cfg)
	newDatabaseListCmd(root, cfg)
	newDatabaseDeleteCmd(root, cfg)
	newDatabaseUsersCreateCmd(usersCmd.Cmd, cfg)
	newDatabaseUsersRetrieveCmd(usersCmd.Cmd, cfg)
	newDatabaseUsersListCmd(usersCmd.Cmd, cfg)
	newDatabaseUsersDeleteCmd(usersCmd.Cmd, cfg)
}

func newDatabaseOperationCmd(parentCmd *cobra.Command, opSpec *OperationSpec, cfg *config.Config, short string) *OperationCmd {
	opCmd := NewOperationCmd(parentCmd, opSpec, cfg)
	opCmd.SuppressOutput = true
	opCmd.Cmd.Short = short
	opCmd.Cmd.Long = databasesLongDescription
	_ = opCmd.Cmd.Flags().MarkHidden("stripe-version")

	if strings.EqualFold(opSpec.Method, http.MethodDelete) {
		_ = opCmd.Cmd.Flags().MarkHidden("confirm")
	}

	return opCmd
}

func newDatabaseCreateCmd(parentCmd *cobra.Command, cfg *config.Config) *cobra.Command {
	opCmd := newDatabaseOperationCmd(parentCmd, &databaseCreateOperationSpec, cfg, "Create a StripeDB instance")
	opCmd.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runDatabaseCreate(cmd, opCmd, args)
	}
	return opCmd.Cmd
}

func newDatabaseRetrieveCmd(parentCmd *cobra.Command, cfg *config.Config) *cobra.Command {
	opCmd := newDatabaseOperationCmd(parentCmd, &databaseRetrieveOperationSpec, cfg, "Retrieve a StripeDB instance")
	opCmd.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runDatabaseRetrieve(cmd, opCmd, args)
	}
	return opCmd.Cmd
}

func newDatabaseListCmd(parentCmd *cobra.Command, cfg *config.Config) *cobra.Command {
	opCmd := newDatabaseOperationCmd(parentCmd, &databaseListOperationSpec, cfg, "List StripeDB instances")
	opCmd.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runDatabaseList(cmd, opCmd, args)
	}
	return opCmd.Cmd
}

func newDatabaseDeleteCmd(parentCmd *cobra.Command, cfg *config.Config) *cobra.Command {
	opCmd := newDatabaseOperationCmd(parentCmd, &databaseDeleteOperationSpec, cfg, "Delete a StripeDB instance")
	var yes bool
	opCmd.Cmd.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation prompt")
	opCmd.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runDatabaseDelete(cmd, opCmd, yes, args)
	}
	return opCmd.Cmd
}

func newDatabaseUsersCreateCmd(parentCmd *cobra.Command, cfg *config.Config) *cobra.Command {
	opCmd := newDatabaseOperationCmd(parentCmd, &databaseUsersCreateOperationSpec, cfg, "Create a StripeDB user")
	opCmd.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runDatabaseUsersCreate(cmd, opCmd, args)
	}
	return opCmd.Cmd
}

func newDatabaseUsersRetrieveCmd(parentCmd *cobra.Command, cfg *config.Config) *cobra.Command {
	opCmd := newDatabaseOperationCmd(parentCmd, &databaseUsersRetrieveOperationSpec, cfg, "Retrieve a StripeDB user")
	opCmd.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runDatabaseUsersRetrieve(cmd, opCmd, args)
	}
	return opCmd.Cmd
}

func newDatabaseUsersListCmd(parentCmd *cobra.Command, cfg *config.Config) *cobra.Command {
	opCmd := newDatabaseOperationCmd(parentCmd, &databaseUsersListOperationSpec, cfg, "List StripeDB users")
	opCmd.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runDatabaseUsersList(cmd, opCmd, args)
	}
	return opCmd.Cmd
}

func newDatabaseUsersDeleteCmd(parentCmd *cobra.Command, cfg *config.Config) *cobra.Command {
	opCmd := newDatabaseOperationCmd(parentCmd, &databaseUsersDeleteOperationSpec, cfg, "Delete a StripeDB user")
	var yes bool
	opCmd.Cmd.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation prompt")
	opCmd.Cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runDatabaseUsersDelete(cmd, opCmd, yes, args)
	}
	return opCmd.Cmd
}

func runDatabaseCreate(cmd *cobra.Command, opCmd *OperationCmd, args []string) error {
	body, err := executeDatabaseOperation(cmd, opCmd, args)
	if err != nil || body == nil {
		return err
	}

	if jsonOutputEnabled(cmd) {
		return writePrettyJSON(cmd, body)
	}

	database, err := decodeDatabase(body)
	if err != nil {
		return err
	}

	user := databaseUser{}
	if database.User != nil {
		user = *database.User
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Created StripeDB instance %s\n", database.ID)
	printDatabaseIndentedMetadataLine(out, "API Version", database.APIVersion)
	printDatabaseIndentedMetadataLine(out, "Mode", databaseModeLabel(database.Livemode))

	fmt.Fprintln(out)
	printDatabaseDetailBlock(out, "Connection details:", []databaseDetailField{
		{Label: "Host", Value: database.Connection.Host},
		{Label: "Username", Value: user.Username},
		{Label: "Password", Value: user.Password},
		{Label: "URL", Value: user.URL},
	})

	status := strings.ToLower(database.Status)
	if status != "" {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "  %s %s. %s stripe databases retrieve %s\n",
			textMuted(out, "Current status:"),
			status,
			textFaint(out, "Check progress with:"),
			database.ID,
		)
	}

	return nil
}

func runDatabaseRetrieve(cmd *cobra.Command, opCmd *OperationCmd, args []string) error {
	body, err := executeDatabaseOperation(cmd, opCmd, args)
	if err != nil || body == nil {
		return err
	}

	if jsonOutputEnabled(cmd) {
		return writePrettyJSON(cmd, body)
	}

	database, err := decodeDatabase(body)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "StripeDB instance %s\n\n", database.ID)
	printDatabaseTable(out, []databaseObject{database})
	return nil
}

func runDatabaseList(cmd *cobra.Command, opCmd *OperationCmd, args []string) error {
	body, err := executeDatabaseOperation(cmd, opCmd, args)
	if err != nil || body == nil {
		return err
	}

	if jsonOutputEnabled(cmd) {
		return writePrettyJSON(cmd, body)
	}

	databases, err := decodeDatabaseList(body)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	printDatabaseListHeading(out, opCmd.Profile)
	printDatabaseTable(out, databases)
	return nil
}

func runDatabaseDelete(cmd *cobra.Command, opCmd *OperationCmd, yes bool, args []string) error {
	if opCmd.DryRun {
		_, err := executeDatabaseOperation(cmd, opCmd, args)
		return err
	}

	if jsonOutputEnabled(cmd) && !yes {
		return fmt.Errorf("--yes is required with --json")
	}

	confirmed, err := confirmDatabaseAction(cmd, yes, "Warning: this will permanently delete your StripeDB instance.", databaseDeleteConfirmationPhrase)
	if err != nil || !confirmed {
		return err
	}

	body, err := executeDatabaseOperation(cmd, opCmd, args)
	if err != nil || body == nil {
		return err
	}

	if jsonOutputEnabled(cmd) {
		return writePrettyJSON(cmd, body)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Deleted StripeDB instance %s\n", args[0])
	return nil
}

func runDatabaseUsersCreate(cmd *cobra.Command, opCmd *OperationCmd, args []string) error {
	body, err := executeDatabaseOperation(cmd, opCmd, args)
	if err != nil || body == nil {
		return err
	}

	if jsonOutputEnabled(cmd) {
		return writePrettyJSON(cmd, body)
	}

	user, err := decodeDatabaseUser(body)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Created StripeDB user %s\n", user.ID)
	printDatabaseIndentedMetadataLine(out, "Username", user.Username)
	printDatabaseIndentedMetadataLine(out, "Mode", databaseModeLabel(user.Livemode))
	fmt.Fprintln(out)
	printDatabaseDetailBlock(out, "Connection details:", []databaseDetailField{
		{Label: "Password", Value: user.Password},
		{Label: "URL", Value: user.URL},
	})
	return nil
}

func runDatabaseUsersRetrieve(cmd *cobra.Command, opCmd *OperationCmd, args []string) error {
	body, err := executeDatabaseOperation(cmd, opCmd, args)
	if err != nil || body == nil {
		return err
	}

	if jsonOutputEnabled(cmd) {
		return writePrettyJSON(cmd, body)
	}

	user, err := decodeDatabaseUser(body)
	if err != nil {
		return err
	}

	printDatabaseUserSection(cmd.OutOrStdout(), args[0], []databaseUser{user})
	return nil
}

func runDatabaseUsersList(cmd *cobra.Command, opCmd *OperationCmd, args []string) error {
	body, err := executeDatabaseOperation(cmd, opCmd, args)
	if err != nil || body == nil {
		return err
	}

	if jsonOutputEnabled(cmd) {
		return writePrettyJSON(cmd, body)
	}

	users, err := decodeDatabaseUserList(body)
	if err != nil {
		return err
	}

	printDatabaseUserSection(cmd.OutOrStdout(), args[0], users)
	return nil
}

func runDatabaseUsersDelete(cmd *cobra.Command, opCmd *OperationCmd, yes bool, args []string) error {
	if opCmd.DryRun {
		_, err := executeDatabaseOperation(cmd, opCmd, args)
		return err
	}

	if jsonOutputEnabled(cmd) && !yes {
		return fmt.Errorf("--yes is required with --json")
	}

	prompt := fmt.Sprintf("Warning: this will permanently remove StripeDB access for user %s.", args[1])
	confirmed, err := confirmDatabaseAction(cmd, yes, prompt, databaseUserDeleteConfirmationText)
	if err != nil || !confirmed {
		return err
	}

	body, err := executeDatabaseOperation(cmd, opCmd, args)
	if err != nil || body == nil {
		return err
	}

	if jsonOutputEnabled(cmd) {
		return writePrettyJSON(cmd, body)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Deleted StripeDB user %s\n", args[1])
	printDatabaseIndentedMetadataLine(out, "StripeDB", args[0])
	return nil
}

func executeDatabaseOperation(cmd *cobra.Command, opCmd *OperationCmd, args []string) ([]byte, error) {
	if err := stripe.ValidateAPIBaseURL(opCmd.APIBaseURL); err != nil {
		return nil, err
	}

	opCmd.Parameters.SetVersion(databaseRequestVersion)

	path := formatURL(opCmd.Path, args)
	requestParams := make(map[string]interface{})
	opCmd.addStringRequestParams(requestParams)
	opCmd.addIntRequestParams(requestParams)
	opCmd.addBoolRequestParams(requestParams)

	if err := opCmd.addArrayRequestParams(requestParams); err != nil {
		return nil, err
	}

	apiKey, apiKeyErr := opCmd.Profile.GetAPIKey(opCmd.Livemode)
	if opCmd.DryRun {
		dryRunKey := apiKey
		if apiKeyErr != nil {
			dryRunKey = ""
		}

		output, err := opCmd.BuildDryRunOutput(dryRunKey, opCmd.APIBaseURL, path, &opCmd.Parameters, requestParams)
		if err != nil {
			return nil, err
		}
		if output.DryRun.Headers == nil {
			output.DryRun.Headers = make(map[string]string)
		}
		output.DryRun.Headers["Stripe-Version"] = databaseRequestVersion

		b, _ := json.MarshalIndent(output, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
		return nil, nil
	}

	if apiKeyErr != nil {
		return nil, apiKeyErr
	}

	body, err := opCmd.MakeRequest(cmd.Context(), apiKey, path, &opCmd.Parameters, requestParams, true, nil)
	return body, err
}

func decodeDatabase(body []byte) (databaseObject, error) {
	var databaseRecord databaseObject
	if err := json.Unmarshal(body, &databaseRecord); err == nil && databaseRecord.ID != "" {
		return databaseRecord, nil
	}

	var envelope databaseEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Database.ID != "" {
		return envelope.Database, nil
	}

	return databaseRecord, json.Unmarshal(body, &databaseRecord)
}

func decodeDatabaseUser(body []byte) (databaseUser, error) {
	var user databaseUser
	if err := json.Unmarshal(body, &user); err == nil && user.ID != "" {
		return user, nil
	}

	var envelope databaseUserEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.DatabaseUser.ID != "" {
		return envelope.DatabaseUser, nil
	}

	return user, json.Unmarshal(body, &user)
}

func decodeDatabaseUserList(body []byte) ([]databaseUser, error) {
	var listResponse dataListEnvelope[databaseUser]
	if err := json.Unmarshal(body, &listResponse); err == nil && listResponse.Data != nil {
		return listResponse.Data, nil
	}

	var users []databaseUser
	if err := json.Unmarshal(body, &users); err == nil {
		return users, nil
	}

	return nil, json.Unmarshal(body, &listResponse)
}

func decodeDatabaseList(body []byte) ([]databaseObject, error) {
	var listResponse dataListEnvelope[databaseObject]
	if err := json.Unmarshal(body, &listResponse); err == nil && listResponse.Data != nil {
		return listResponse.Data, nil
	}

	var databases []databaseObject
	if err := json.Unmarshal(body, &databases); err == nil {
		return databases, nil
	}

	return nil, json.Unmarshal(body, &listResponse)
}

func jsonOutputEnabled(cmd *cobra.Command) bool {
	enabled, err := cmd.Flags().GetBool(databaseJSONFlagName)
	return err == nil && enabled
}

func writePrettyJSON(cmd *cobra.Command, body []byte) error {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil
	}

	var out bytes.Buffer
	if err := json.Indent(&out, body, "", "  "); err != nil {
		return err
	}
	out.WriteByte('\n')

	_, err := cmd.OutOrStdout().Write(out.Bytes())
	return err
}

func printDatabaseTable(out io.Writer, databases []databaseObject) {
	headers := []string{"ID", "Host", "Status", "Created", "Mode", "API Version"}
	rows := make([][]string, 0, len(databases))

	for _, database := range databases {
		rows = append(rows, []string{
			database.ID,
			database.Connection.Host,
			strings.ToLower(database.Status),
			databaseRelativeTime(database.Created),
			databaseModeLabel(database.Livemode),
			database.APIVersion,
		})
	}

	printAlignedTable(out, headers, rows, "No StripeDB instances found.", func(i int, value string) string {
		if i%2 == 1 {
			return textMuted(out, value)
		}
		return value
	})
}

func printDatabaseUserSection(out io.Writer, databaseID string, users []databaseUser) {
	fmt.Fprintf(out, "StripeDB users for %s\n\n", databaseID)
	printDatabaseUserTable(out, users)
}

func printDatabaseUserTable(out io.Writer, users []databaseUser) {
	headers := []string{"ID", "Username", "Created", "Mode"}
	rows := make([][]string, 0, len(users))

	for _, user := range users {
		rows = append(rows, []string{
			user.ID,
			user.Username,
			databaseRelativeTime(user.Created),
			databaseModeLabel(user.Livemode),
		})
	}

	printAlignedTable(out, headers, rows, "No StripeDB users found.", func(i int, value string) string {
		if i >= 2 {
			return textMuted(out, value)
		}
		return value
	})
}

func textBold(out io.Writer, text string) string {
	return ansi.Color(out).Bold(text).String()
}

func textMuted(out io.Writer, text string) string {
	return ansi.Color(out).BrightBlack(text).String()
}

func textFaint(out io.Writer, text string) string {
	return ansi.Color(out).Faint(text).String()
}

func printDatabaseIndentedMetadataLine(out io.Writer, label, value string) {
	if value == "" {
		return
	}

	fmt.Fprintf(out, "  %s %s\n", textMuted(out, label+":"), value)
}

func printDatabaseListHeading(out io.Writer, profile *config.Profile) {
	if profile != nil {
		accountID, err := profile.GetAccountID()
		if err == nil && accountID != "" {
			fmt.Fprintf(out, "StripeDB instances for account %s\n\n", accountID)
			return
		}
	}

	fmt.Fprintln(out, "StripeDB instances")
	fmt.Fprintln(out)
}

func printDatabaseDetailBlock(out io.Writer, title string, fields []databaseDetailField) {
	labelWidth := 0
	visibleFieldCount := 0
	for _, field := range fields {
		if field.Value == "" {
			continue
		}

		visibleFieldCount++
		if width := len(field.Label) + 1; width > labelWidth {
			labelWidth = width
		}
	}

	if visibleFieldCount == 0 {
		return
	}

	fmt.Fprintln(out, textBold(out, title))
	for _, field := range fields {
		if field.Value == "" {
			continue
		}

		label := field.Label + ":"
		padding := labelWidth - len(label) + 2
		fmt.Fprint(out, "  ")
		fmt.Fprint(out, textMuted(out, label))
		fmt.Fprint(out, strings.Repeat(" ", padding))
		fmt.Fprintln(out, field.Value)
	}
}

func printAlignedTable(out io.Writer, headers []string, rows [][]string, emptyMessage string, render func(int, string) string) {
	if len(rows) == 0 {
		fmt.Fprintln(out, textMuted(out, emptyMessage))
		return
	}

	widths := tableColumnWidths(headers, rows)
	printDatabaseTableRow(out, headers, widths, func(_ int, value string) string {
		return textBold(out, value)
	})
	printDatabaseTableRow(out, tableSeparators(widths), widths, func(_ int, value string) string {
		return textFaint(out, value)
	})
	for _, row := range rows {
		printDatabaseTableRow(out, row, widths, render)
	}
}

func tableColumnWidths(headers []string, rows [][]string) []int {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}

	for _, row := range rows {
		for i, value := range row {
			if len(value) > widths[i] {
				widths[i] = len(value)
			}
		}
	}

	return widths
}

func tableSeparators(widths []int) []string {
	separators := make([]string, len(widths))
	for i, width := range widths {
		separators[i] = strings.Repeat("-", width)
	}
	return separators
}

func printDatabaseTableRow(out io.Writer, values []string, widths []int, render func(int, string) string) {
	for i, value := range values {
		fmt.Fprint(out, render(i, value))
		if i == len(values)-1 {
			continue
		}

		padding := widths[i] - len(value) + 2
		fmt.Fprint(out, strings.Repeat(" ", padding))
	}

	fmt.Fprintln(out)
}

func databaseModeLabel(livemode bool) string {
	if livemode {
		return "live"
	}

	return "test"
}

func databaseRelativeTime(raw string) string {
	if raw == "" {
		return ""
	}

	createdAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}

	now := databaseNow().UTC()
	createdAt = createdAt.UTC()
	if createdAt.After(now) {
		return "now"
	}

	if years := fullYearsBetween(createdAt, now); years > 0 {
		return fmt.Sprintf("%dy", years)
	}

	if months := fullMonthsBetween(createdAt, now); months > 0 {
		return fmt.Sprintf("%dmo", months)
	}

	diff := now.Sub(createdAt)
	if days := int(diff / (24 * time.Hour)); days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	if hours := int(diff / time.Hour); hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	if minutes := int(diff / time.Minute); minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}

	return "now"
}

func fullYearsBetween(earlier, later time.Time) int {
	years := later.Year() - earlier.Year()
	if earlier.AddDate(years, 0, 0).After(later) {
		years--
	}
	if years < 0 {
		return 0
	}
	return years
}

func fullMonthsBetween(earlier, later time.Time) int {
	months := (later.Year()-earlier.Year())*12 + int(later.Month()-earlier.Month())
	if earlier.AddDate(0, months, 0).After(later) {
		months--
	}
	if months < 0 {
		return 0
	}
	return months
}

func confirmDatabaseAction(cmd *cobra.Command, autoConfirm bool, warning, confirmationPhrase string) (bool, error) {
	if autoConfirm {
		return true, nil
	}

	out := cmd.OutOrStdout()
	color := ansi.Color(out)
	fmt.Fprintln(out, color.Yellow(warning).String())
	if confirmationPhrase != "" {
		fmt.Fprintf(out, "%s %s %s\n", textFaint(out, "Type"), textBold(out, confirmationPhrase), textFaint(out, "to continue."))
	} else {
		fmt.Fprintf(out, "%s %s %s\n", textFaint(out, "Type"), textBold(out, "y"), textFaint(out, "to continue, or press Enter to cancel."))
	}
	fmt.Fprint(out, textMuted(out, "> "))

	reader := bufio.NewReader(cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}

	trimmed := strings.TrimSpace(input)
	confirmed := false
	if confirmationPhrase != "" {
		confirmed = strings.EqualFold(trimmed, confirmationPhrase)
	} else {
		trimmed = strings.ToLower(trimmed)
		confirmed = trimmed == "y" || trimmed == "yes"
	}

	if confirmed {
		fmt.Fprintln(out)
	}
	return confirmed, nil
}
