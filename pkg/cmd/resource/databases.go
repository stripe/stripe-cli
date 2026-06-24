package resource

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/cmdutil"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

const (
	databaseJSONFlagName               = "json"
	databaseRequestVersion             = "unsafe-development"
	databaseDeleteConfirmationPhrase   = "delete database"
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
	Name       string             `json:"name"` // TODO: remove placeholder once API ships display_name
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
	var sp *spinner.Spinner
	if !jsonOutputEnabled(cmd) {
		sp = ansi.StartNewSpinner("Creating StripeDB instance...", cmd.ErrOrStderr())
	}
	body, err := executeDatabaseOperation(cmd, opCmd, args)
	if sp != nil {
		ansi.StopSpinner(sp, "", cmd.ErrOrStderr())
	}

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

	// Leading blank line to match design spec
	fmt.Fprintln(out)

	// Progress steps (static, shown after API responds)
	checkMark := "✓"
	circle := "○"
	if !databaseUnicodeSupported() {
		checkMark = "[done]"
		circle = "o"
	}
	steps := []struct {
		glyph string
		label string
		color func(io.Writer, string) string
	}{
		{checkMark, "Provisioned database", textGreen},
		{checkMark, "Generated credentials", textGreen},
		{circle, "Syncing data...", textYellow},
	}
	for _, s := range steps {
		fmt.Fprintf(out, "  %s %s\n", s.color(out, s.glyph), s.label)
		if sp != nil {
			time.Sleep(30 * time.Millisecond)
		}
	}
	fmt.Fprintln(out)

	fmt.Fprintf(out, "Created StripeDB instance %s (%s)\n",
		textCyan(out, databaseTruncateID(database.ID)),
		databaseDisplayName(database),
	)
	printDatabaseIndentedMetadataLine(out, "API Version", database.APIVersion)
	printDatabaseIndentedMetadataLine(out, "Mode", databaseModeLabel(database.Livemode))

	fmt.Fprintln(out)
	printDatabaseDetailBlock(out, "Connection details:", []databaseDetailField{
		{Label: "Host", Value: database.Connection.Host},
		{Label: "Username", Value: user.Username},
		{Label: "Password", Value: user.Password},
		{Label: "URL", Value: user.URL},
	})

	if user.Password != "" {
		fmt.Fprintln(out)
		fmt.Fprintln(out, textYellow(out, databaseWarningGlyph()+" Save this password now — it will not be shown again."))
	}

	fmt.Fprintln(out)
	dashURL := databaseDashboardURL(database.ID, database.Livemode)
	fmt.Fprintf(out, "  %s %s\n", textBoldCyan(out, "Dashboard:"), ansi.Linkify(dashURL, dashURL, out))

	if database.Status != "" {
		fmt.Fprintln(out)
		statusVal := databaseStatusColored(out, database.Status, databaseStatusLabel(database.Status))
		fmt.Fprintf(out, "  %s %s. %s\n  %s\n",
			textMuted(out, "Current status:"),
			statusVal,
			textFaint(out, "Check progress with:"),
			textCyan(out, "stripe databases retrieve "+database.ID),
		)
	}

	fmt.Fprintln(out)
	// "Privacy Policy" and "Preview Terms" highlighted white; surrounding text faint
	fmt.Fprintf(out, "%s %s %s %s\n",
		textFaint(out, "By creating a database, you agree to the"),
		textBold(out, "Privacy Policy"),
		textFaint(out, "and"),
		textBold(out, "Preview Terms."),
	)
	privacyURL := "https://stripe.com/privacy"
	termsURL := "https://stripe.com/stripe-database-preview-terms"
	fmt.Fprintf(out, "  %s\n", ansi.Linkify(privacyURL, privacyURL, out))
	fmt.Fprintf(out, "  %s\n", ansi.Linkify(termsURL, termsURL, out))

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
	fmt.Fprintln(out)
	// Show name + full dimmed ID once API ships display_name; fall back to full ID until then.
	if database.Name != "" {
		fmt.Fprintf(out, "%s %s\n  %s\n\n",
			textBold(out, "StripeDB instance"),
			database.Name,
			textMuted(out, database.ID),
		)
	} else {
		fmt.Fprintf(out, "%s %s\n\n",
			textBold(out, "StripeDB instance"),
			database.ID,
		)
	}

	// Block 1: instance metadata (no section header)
	printDatabaseDetailBlock(out, "", []databaseDetailField{
		{Label: "Status", Value: databaseStatusColored(out, database.Status, databaseStatusCell(database.Status))},
		{Label: "API Version", Value: database.APIVersion},
		{Label: "Mode", Value: databaseModeLabel(database.Livemode)},
		{Label: "Created", Value: databaseRelativeTimeAgo(database.Created)},
	})

	fmt.Fprintln(out)

	// Block 2: connection details
	printDatabaseDetailBlock(out, "Connection details:", []databaseDetailField{
		{Label: "Host", Value: database.Connection.Host},
		{Label: "Port", Value: func() string {
			if database.Connection.Port == 0 {
				return ""
			}
			return fmt.Sprintf("%d", database.Connection.Port)
		}()},
		{Label: "Database", Value: database.Connection.DatabaseName},
	})

	fmt.Fprintln(out)
	dashURL := databaseDashboardURL(database.ID, database.Livemode)
	fmt.Fprintf(out, "%s\n  %s\n",
		textMuted(out, "View in Dashboard:"),
		ansi.Linkify(dashURL, dashURL, out),
	)
	fmt.Fprintln(out)
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
	fmt.Fprintln(out)
	printDatabaseListHeading(out, opCmd.Profile)
	printDatabaseTable(out, databases)
	fmt.Fprintln(out)
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

	dbName := databaseDisplayName(databaseObject{ID: args[0]})
	warning := fmt.Sprintf("%s Warning: this will permanently delete %s (%s).",
		databaseWarningGlyph(), dbName, databaseTruncateID(args[0]))
	confirmed, err := confirmDatabaseAction(cmd, yes, warning, databaseDeleteConfirmationPhrase)
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
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s %s\n",
		textGreen(out, "Deleted "+dbName),
		textMuted(out, "("+databaseTruncateID(args[0])+")"),
	)
	fmt.Fprintln(out)
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
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s\n", textGreen(out, "Created StripeDB user "+user.ID))
	printDatabaseIndentedMetadataLine(out, "Username", user.Username)
	printDatabaseIndentedMetadataLine(out, "Mode", databaseModeLabel(user.Livemode))
	fmt.Fprintln(out)
	printDatabaseDetailBlock(out, "Connection details:", []databaseDetailField{
		{Label: "Password", Value: user.Password},
		{Label: "URL", Value: user.URL},
	})

	if user.Password != "" {
		fmt.Fprintln(out)
		fmt.Fprintln(out, textYellow(out, databaseWarningGlyph()+" Save this password now — it will not be shown again."))
	}

	fmt.Fprintln(out)
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

	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	dbName := databaseDisplayName(databaseObject{ID: args[0]})
	fmt.Fprintf(out, "StripeDB user %s\n  %s\n\n",
		user.ID,
		textMuted(out, dbName+" ("+args[0]+")"),
	)
	printDatabaseDetailBlock(out, "Details:", []databaseDetailField{
		{Label: "Username", Value: user.Username},
		{Label: "Mode", Value: databaseModeLabel(user.Livemode)},
		{Label: "Created", Value: databaseRelativeTimeAgo(user.Created)},
		{Label: "Database", Value: args[0]},
	})
	fmt.Fprintln(out)
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

	prompt := fmt.Sprintf("%s Warning: this will permanently remove StripeDB access for user %s.", databaseWarningGlyph(), args[1])
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
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s\n", textGreen(out, "Deleted "+args[1]))
	fmt.Fprintln(out)
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
	// TODO: uncomment Name column when API ships the display_name field
	// {
	// 	header: "Name",
	// 	value:  func(i int) string { return databaseDisplayName(databases[i]) },
	// 	style:  func(o io.Writer, _ int, v string) string { return textCyan(o, v) },
	// },
	printTable(out, []tableColumn{
		{
			header: "ID",
			value:  func(i int) string { return databases[i].ID },
			style:  func(o io.Writer, _ int, v string) string { return textMuted(o, v) },
		},
		{
			header: "Status",
			value:  func(i int) string { return databaseStatusCell(databases[i].Status) },
			style: func(o io.Writer, i int, v string) string {
				return databaseStatusColored(o, databases[i].Status, v)
			},
		},
		{
			header: "Created",
			value:  func(i int) string { return databaseRelativeTime(databases[i].Created) },
			style:  func(o io.Writer, _ int, v string) string { return textMuted(o, v) },
		},
		{
			header: "Mode",
			value:  func(i int) string { return databaseModeLabel(databases[i].Livemode) },
		},
		{
			header: "API Version",
			value:  func(i int) string { return databases[i].APIVersion },
			style:  func(o io.Writer, _ int, v string) string { return textMuted(o, v) },
		},
	}, len(databases), "No StripeDB instances found.")
}

func printDatabaseUserSection(out io.Writer, databaseID string, users []databaseUser) {
	fmt.Fprintln(out)
	fmt.Fprintf(out, "StripeDB users for %s\n\n", databaseID)
	printDatabaseUserTable(out, users)
	fmt.Fprintln(out)
}

func printDatabaseUserTable(out io.Writer, users []databaseUser) {
	printTable(out, []tableColumn{
		{
			header: "Username",
			value:  func(i int) string { return users[i].Username },
			style:  func(o io.Writer, _ int, v string) string { return textCyan(o, v) },
		},
		{
			header: "ID",
			value:  func(i int) string { return users[i].ID },
			style:  func(o io.Writer, _ int, v string) string { return textMuted(o, v) },
		},
		{
			header: "Created",
			value:  func(i int) string { return databaseRelativeTime(users[i].Created) },
			style:  func(o io.Writer, _ int, v string) string { return textMuted(o, v) },
		},
		{
			header: "Mode",
			value:  func(i int) string { return databaseModeLabel(users[i].Livemode) },
			style:  func(o io.Writer, _ int, v string) string { return textMuted(o, v) },
		},
	}, len(users), "No StripeDB users found.")
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

func textCyan(out io.Writer, text string) string {
	return ansi.Color(out).Cyan(text).String()
}

func textGreen(out io.Writer, text string) string {
	return ansi.Color(out).Green(text).String()
}

func textYellow(out io.Writer, text string) string {
	return ansi.Color(out).Yellow(text).String()
}

func textBoldCyan(out io.Writer, text string) string {
	c := ansi.Color(out)
	return c.BrightCyan(c.Bold(text)).String()
}

func textRed(out io.Writer, text string) string {
	return ansi.Color(out).Red(text).String()
}

// databaseWarningGlyph returns ⚠ on Unicode-capable terminals, ! otherwise.
func databaseWarningGlyph() string {
	if databaseUnicodeSupported() {
		return "⚠"
	}
	return "!"
}

// databaseRelativeTimeAgo wraps databaseRelativeTime and appends " ago" for
// non-empty, non-"now" values (e.g. "5d ago"). Returns "just now" for "now".
func databaseRelativeTimeAgo(raw string) string {
	rel := databaseRelativeTime(raw)
	switch rel {
	case "", "now":
		return "just now"
	default:
		return rel + " ago"
	}
}

// databaseUnicodeSupported reports whether the current terminal renders Unicode glyphs correctly.
// On Windows, only Windows Terminal (WT_SESSION) and ConEmu (ConEmuPID) are known-safe.
func databaseUnicodeSupported() bool {
	if runtime.GOOS == "windows" {
		_, wt := os.LookupEnv("WT_SESSION")
		_, ce := os.LookupEnv("ConEmuPID")
		return wt || ce
	}
	return true
}

// visualWidth returns the display width of s after stripping ANSI escape sequences.
// Uses rune count rather than byte count so multibyte glyphs (●, ─, ✓) measure as 1.
func visualWidth(s string) int {
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	i := 0
	for i < len(runes) {
		if runes[i] == '\x1b' && i+1 < len(runes) {
			switch runes[i+1] {
			case '[': // CSI: ESC [ ... letter
				i += 2
				for i < len(runes) && (runes[i] < 'A' || runes[i] > 'Z') && (runes[i] < 'a' || runes[i] > 'z') {
					i++
				}
				if i < len(runes) {
					i++
				}
				continue
			case ']': // OSC: ESC ] ... ESC backslash
				i += 2
				for i < len(runes) {
					if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '\\' {
						i += 2
						break
					}
					i++
				}
				continue
			}
		}
		out = append(out, runes[i])
		i++
	}
	return len(out)
}

// databaseStatusLabel remaps internal API status values to user-facing labels.
// Unknown statuses are title-cased and passed through unchanged.
func databaseStatusLabel(rawStatus string) string {
	switch strings.ToLower(rawStatus) {
	case "ready":
		return "Active"
	case "backfilling":
		return "Backfilling"
	default:
		if rawStatus == "" {
			return ""
		}
		lower := strings.ToLower(rawStatus)
		return strings.ToUpper(lower[:1]) + lower[1:]
	}
}

// databaseStatusCell returns the plain-text display string for a status cell (glyph + label).
// This is used for column-width calculation; color is applied separately at render time.
func databaseStatusCell(rawStatus string) string {
	label := databaseStatusLabel(rawStatus)
	if databaseUnicodeSupported() {
		switch strings.ToLower(rawStatus) {
		case "ready":
			return "● " + label
		case "error":
			return "✗ " + label
		default:
			return "○ " + label
		}
	}
	switch strings.ToLower(rawStatus) {
	case "ready":
		return "* " + label
	case "error":
		return "x " + label
	default:
		return "o " + label
	}
}

// databaseStatusColored wraps a status cell value with the appropriate terminal color.
func databaseStatusColored(out io.Writer, rawStatus, cellValue string) string {
	switch strings.ToLower(rawStatus) {
	case "ready":
		return textGreen(out, cellValue)
	case "backfilling":
		return textYellow(out, cellValue)
	case "error":
		return textRed(out, cellValue)
	default:
		return cellValue
	}
}

// databaseTruncateID shortens a database ID by truncating the middle portion.
// "db_1XyZ2aBcDeFgHiJkLmN8pQr" → "db_1Xy...N8pQr"
func databaseTruncateID(id string) string {
	const keep = 6
	if len(id) <= keep*2+3 {
		return id
	}
	return id[:keep] + "..." + id[len(id)-keep:]
}

// databaseDisplayName returns the human-readable display name for a database.
// TODO: use db.Name once the API ships the display_name field; remove placeholder.
func databaseDisplayName(db databaseObject) string {
	if db.Name != "" {
		return db.Name
	}
	return "StripeDB Instance"
}

// databaseDashboardURL returns the Dashboard URL for a database.
// Test-mode databases use the /test/ path prefix.
func databaseDashboardURL(id string, livemode bool) string {
	if livemode {
		return fmt.Sprintf("https://dashboard.stripe.com/data-management/databases/%s", id)
	}
	return fmt.Sprintf("https://dashboard.stripe.com/test/data-management/databases/%s", id)
}

func printDatabaseIndentedMetadataLine(out io.Writer, label, value string) {
	if value == "" {
		return
	}

	fmt.Fprintf(out, "  %s %s\n", textBoldCyan(out, label+":"), value)
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

	if title != "" {
		fmt.Fprintln(out, textBold(out, title))
	}
	for _, field := range fields {
		if field.Value == "" {
			continue
		}

		label := field.Label + ":"
		padding := labelWidth - len(label) + 2
		fmt.Fprint(out, "  ")
		fmt.Fprint(out, textBoldCyan(out, label))
		fmt.Fprint(out, strings.Repeat(" ", padding))
		fmt.Fprintln(out, field.Value)
	}
}

// tableColumn defines a single column in a CLI table.
// value extracts the plain display string (used for width calculation).
// style optionally wraps the plain value with ANSI formatting.
// If style is nil the plain value is printed as-is.
type tableColumn struct {
	header string
	value  func(rowIdx int) string
	style  func(out io.Writer, rowIdx int, value string) string
}

func printTable(out io.Writer, columns []tableColumn, rowCount int, emptyMessage string) {
	if rowCount == 0 {
		fmt.Fprintln(out, textMuted(out, emptyMessage))
		return
	}

	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.header
	}
	rows := make([][]string, rowCount)
	for i := 0; i < rowCount; i++ {
		row := make([]string, len(columns))
		for j, col := range columns {
			row[j] = col.value(i)
		}
		rows[i] = row
	}

	widths := make([]int, len(columns))
	for i, h := range headers {
		widths[i] = visualWidth(h)
	}
	for _, row := range rows {
		for i, v := range row {
			if w := visualWidth(v); w > widths[i] {
				widths[i] = w
			}
		}
	}

	sep := "-"
	if databaseUnicodeSupported() {
		sep = "─"
	}

	for i, h := range headers {
		fmt.Fprint(out, textBold(out, h))
		if i < len(headers)-1 {
			fmt.Fprint(out, strings.Repeat(" ", widths[i]-visualWidth(h)+2))
		}
	}
	fmt.Fprintln(out)

	for i, w := range widths {
		fmt.Fprint(out, textFaint(out, strings.Repeat(sep, w)))
		if i < len(widths)-1 {
			fmt.Fprint(out, strings.Repeat(" ", 2))
		}
	}
	fmt.Fprintln(out)

	for rowIdx, row := range rows {
		ri := rowIdx
		for i, v := range row {
			rendered := v
			if columns[i].style != nil {
				rendered = columns[i].style(out, ri, v)
			}
			fmt.Fprint(out, rendered)
			if i < len(row)-1 {
				fmt.Fprint(out, strings.Repeat(" ", widths[i]-visualWidth(v)+2))
			}
		}
		fmt.Fprintln(out)
	}
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
	fmt.Fprintln(out)
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

	return confirmed, nil
}
