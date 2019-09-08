package samples

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"github.com/manifoldco/promptui"
	"github.com/otiai10/copy"
	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/git"
	"github.com/stripe/stripe-cli/pkg/stripeauth"
)

var fileEnvRegexp = map[string]*regexp.Regexp{
	".java": regexp.MustCompile(`String ENV_PATH = "(.*)";`),
	".js":   regexp.MustCompile(`const envPath = resolve\(__dirname, "(.*)\.env"\);`),
	".php":  regexp.MustCompile(`\$ENV_PATH = '(.*)';`),
	".rb":   regexp.MustCompile(`ENV_PATH = '(.*)\.env'\.freeze`),
}

var fileEnvTemplate = map[string]func(string) string{
	".java": javaPath,
	".js":   jsPath,
	".php":  phpPath,
	".rb":   rbPath,
}

var languageDisplayNames = map[string]string{
	"java":   "Java",
	"node":   "Node",
	"python": "Python",
	"php":    "PHP",
	"ruby":   "Ruby",
}

var displayNameLanguages = reverseStringMap(languageDisplayNames)

func javaPath(path string) string {
	return fmt.Sprintf(`String ENV_PATH = "%s/";`, path)
}

func jsPath(path string) string {
	return fmt.Sprintf(`const envPath = resolve(__dirname, "%s/.env");`, path)
}

func phpPath(path string) string {
	// Need $$ here because otherwise $FOO is treated as an interpretted variable
	return fmt.Sprintf(`$$ENV_PATH = '%s';`, path)
}

func rbPath(path string) string {
	return fmt.Sprintf(`ENV_PATH = '%s/.env'.freeze`, path)
}

func reverseStringMap(m map[string]string) map[string]string {
	rm := make(map[string]string, len(m))

	for key, value := range m {
		rm[value] = key
	}

	return rm
}

// Samples stores the information for the selected sample in addition to the
// selected configuration option to copy over
type Samples struct {
	Config *config.Config
	Fs     afero.Fs
	Git    git.Interface

	name string

	// source repository to clone from
	repo string

	// Available integrations
	integrations  []string
	isIntegration bool

	// Available languages
	languages []string

	//selected integrations. We store selected integration as an array
	// in case they pick more than one
	integration []string

	// selected language
	language string
}

// Initialize get the sample ready for the user to copy. It:
// 1. creates the sample cache folder if it doesn't exist
// 2. store the path of the locale cache folder for later use
// 3. if the selected app does not exist in the local cache folder, clone it
// 4. if the selected app does exist in the local cache folder, pull changes
// 5. see if there are different integrations available for the sample
// 6. see what languages the sample is available in
func (s *Samples) Initialize(app string) error {
	s.name = app

	appPath, err := s.appCacheFolder(app)
	if err != nil {
		return err
	}

	// We still set the repo path here. There are some failure cases
	// that we can still work with (like no updates or repo already exists)
	s.repo = appPath

	if _, err := s.Fs.Stat(appPath); os.IsNotExist(err) {
		err = s.Git.Clone(appPath, List[app].GitRepo())
		if err != nil {
			return err
		}
	} else {
		err := s.Git.Pull(appPath)
		if err != nil {
			return err
		}
	}

	// Samples can have multiple integration types, each of which will have its
	// own client/server implementation. For example, the adding sales tax
	// sample, has a manual confirmation and automatic confirmation integration.
	// These integrations are stored as folders in the top-level of the sample.
	// Since much of the sample setup logic is going to be dependent on the
	// structure of the sample, we want to check for whether there are
	// integrations upfront.
	err = s.checkForIntegrations()
	if err != nil {
		return err
	}

	// Once we've pulled the integration, we want to check what languages are
	// supported so that we can ask the user which language they want to copy.
	err = s.loadLanguages()
	if err != nil {
		return err
	}

	return nil
}

// checkForIntegrations scans the sample to see if there are different
// integration option available. Integratios are the different ways to build
// the specific sample, for example if it uses charges or payment intents
// would be two separate integrations.
//
// A sample's folder structure will either contain "client" and "server"
// folders in its top-level or it'll have folders that each contain a different
// integration. This function scans to see if there is a "server" folder in the
// top level and uses that to determine if there are integrations.
func (s *Samples) checkForIntegrations() error {
	folders, err := s.GetFolders(s.repo)
	if err != nil {
		return err
	}

	if !folderSearch(folders, "server") {
		s.integrations = folders
		s.isIntegration = true
		return nil
	}

	s.isIntegration = false
	return nil
}

// Each sample will have specific languages that it supports. Right now, there
// is a goal to support java, node, php, python, and ruby for all our samples.
// We did not hard code those to avoid having to release a CLI update if we
// ever add new language support.
//
// Samples do not release until all integrations have all supported languages
// built out. With that, we can simply check the languages supported in any
// folder and assume that all will have the same languages.
func (s *Samples) loadLanguages() error {
	var err error

	if s.isIntegration {
		// The same languages will be supported by all integrations in a repo so we can
		// rely on only checking the first
		s.languages, err = s.GetFolders(filepath.Join(s.repo, s.integrations[0], "server"))
	} else {
		s.languages, err = s.GetFolders(filepath.Join(s.repo, "server"))
	}

	if err != nil {
		return err
	}

	return nil
}

// SelectOptions prompts the user to select the integration they want to use
// (if available) and the language they want the integration to be.
func (s *Samples) SelectOptions() error {
	var err error

	if s.isIntegration {
		s.integration, err = integrationSelectPrompt(s.integrations)
		if err != nil {
			return err
		}
	}

	s.language, err = languageSelectPrompt(s.languages)
	if err != nil {
		return err
	}

	fmt.Println("Setting up", ansi.Bold(s.name))

	return nil
}

// Copy will copy all of the files from the selected configuration above oves.
// This has a few different behaviors, depending on the configuration.
// Ultimately, we want the user to do as minimal of folder traversing as
// possible. What we want to end up with is:
//
// |- example-sample/
// +-- client/
// +-- server/
// +-- readme.md
// +-- ...
// `-- .env.example
//
// The behavior here is:
// * If there are no integrations available, copy the top-level files, the
//   client folder, and the selected language inside of the server folder to
//   the server top-level (example above)
// * If the user selects 1 integration, mirror the structure above for the
//   selected integration (example above)
// * If they selected >1 integration, we want the same structure above but
//   replicated once per selected in integration.
func (s *Samples) Copy(target string) error {
	// The condition for the loop starts as true since we will always want to
	// process at least once.
	for i := 0; true; i++ {
		integration := s.destinationName(i)

		serverSource := filepath.Join(s.repo, integration, "server", s.language)
		clientSource := filepath.Join(s.repo, integration, "client")
		filesSource, err := s.GetFiles(filepath.Join(s.repo, integration))
		if err != nil {
			return err
		}

		serverDestination := s.destinationPath(target, integration, "server")
		clientDestination := s.destinationPath(target, integration, "client")

		err = copy.Copy(serverSource, serverDestination)
		if err != nil {
			return err
		}
		err = copy.Copy(clientSource, clientDestination)
		if err != nil {
			return err
		}

		// This copies all top-level files specific to integrations
		for _, file := range filesSource {
			err = copy.Copy(filepath.Join(s.repo, integration, file), filepath.Join(s.destinationPath(target, integration, ""), file))
			if err != nil {
				return err
			}
		}

		if i >= len(s.integration)-1 {
			break
		}
	}

	// This copies all top-level files specific to the entire sample repo
	filesSource, err := s.GetFiles(s.repo)
	if err != nil {
		return err
	}
	for _, file := range filesSource {
		err = copy.Copy(filepath.Join(s.repo, file), filepath.Join(target, file))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Samples) findServerFiles(target string) ([]string, error) {
	var files []string
	err := afero.Walk(s.Fs, target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.Contains(path, "server") {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return []string{}, err
	}

	return files, nil
}

// PointToDotEnv searches through the recently copied files for references to
// `../../..` and changes it to be one (or two) levels removed. Reason for this
// is that as part of configuring the sample, we'll remove 1-2 levels of the
// folder hierarchy so we need to adjust the references we point to.
func (s *Samples) PointToDotEnv(target string) error {
	serverFiles, err := s.findServerFiles(target)
	if err != nil {
		return err
	}

	for _, file := range serverFiles {
		// There are a bunch of files we don't want to process so skip them
		if !fileWhitelist(file) {
			continue
		}

		data, err := afero.ReadFile(s.Fs, file)
		if err != nil {
			return err
		}

		content := string(data)

		dotPath := ".."
		if s.isIntegration && len(s.integration) >= 2 {
			dotPath = "../.."
		}

		regex := fileEnvRegexp[filepath.Ext(file)]
		tmpl := fileEnvTemplate[filepath.Ext(file)]
		updated := regex.ReplaceAllString(content, tmpl(dotPath))

		err = afero.WriteFile(s.Fs, file, []byte(updated), 0)
		if err != nil {
			return err
		}
	}

	return nil
}

// ConfigureDotEnv takes the .env.example from the provided location and
// modifies it to automatically configure it for the users settings
func (s *Samples) ConfigureDotEnv(sampleLocation string) error {
	// .env.example file will always be at the project root
	exFile := filepath.Join(sampleLocation, ".env.example")

	file, err := s.Fs.Open(exFile)
	if err != nil {
		return err
	}

	dotenv, err := godotenv.Parse(file)
	if err != nil {
		return err
	}

	publishableKey := s.Config.Profile.GetPublishableKey()
	if publishableKey == "" {
		return fmt.Errorf("we could not set the publishable key in the .env file; please set this manually or login again to set it automatically next time")
	}

	apiKey, err := s.Config.Profile.GetAPIKey(false)
	if err != nil {
		return err
	}
	deviceName, err := s.Config.Profile.GetDeviceName()
	if err != nil {
		return err
	}

	authClient := stripeauth.NewClient(apiKey, nil)
	authSession, err := authClient.Authorize(deviceName, "webhooks", nil)
	if err != nil {
		return err
	}

	dotenv["STRIPE_PUBLIC_KEY"] = publishableKey
	dotenv["STRIPE_SECRET_KEY"] = apiKey
	dotenv["STRIPE_WEBHOOK_SECRET"] = authSession.Secret
	dotenv["STATIC_DIR"] = "../client"

	envFile := filepath.Join(sampleLocation, ".env")
	err = godotenv.Write(dotenv, envFile)
	if err != nil {
		return err
	}

	return nil
}

func (s *Samples) destinationName(i int) string {
	if len(s.integration) > 0 && s.isIntegration {
		if s.integration[0] == "all" {
			return s.integrations[i]
		}

		return s.integration[i]
	}

	return ""
}

// Cleanup performs cleanup for the recently created sample
func (s *Samples) Cleanup(name string) error {
	fmt.Println("Cleaning up...")

	return s.delete(name)
}

func (s *Samples) destinationPath(target string, integration string, folder string) string {
	if len(s.integration) <= 1 {
		return filepath.Join(target, folder)
	}

	return filepath.Join(target, integration, folder)
}

func selectOptions(template, label string, options []string) (string, error) {
	templates := &promptui.SelectTemplates{
		Selected: ansi.Faint(fmt.Sprintf("Selected %s: {{ . | bold }} ", template)),
	}

	prompt := promptui.Select{
		Label:     label,
		Items:     options,
		Templates: templates,
	}

	_, result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

func languageSelectPrompt(languages []string) (string, error) {
	var displayLangs []string
	for _, lang := range languages {
		if val, ok := languageDisplayNames[lang]; ok {
			displayLangs = append(displayLangs, val)
		} else {
			displayLangs = append(displayLangs, lang)
		}
	}

	selected, err := selectOptions("language", "What language would you like to use", displayLangs)
	if err != nil {
		return "", err
	}

	if val, ok := displayNameLanguages[selected]; ok {
		return val, nil
	}

	return selected, nil
}

func integrationSelectPrompt(integrations []string) ([]string, error) {
	selected, err := selectOptions("integration", "What type of integration would you like to use", append(integrations, "all"))
	if err != nil {
		return []string{}, err
	}

	if selected == "all" {
		return integrations, nil
	}

	return []string{selected}, nil
}

func fileWhitelist(path string) bool {
	// NOTE: we're not changing `.py` files because the Python dotenv library
	// is able to recursively search up directory trees, so it'll correctly
	// find our .env file
	return strings.HasSuffix(path, ".java") ||
		strings.HasSuffix(path, ".php") ||
		strings.HasSuffix(path, ".js") ||
		strings.HasSuffix(path, ".rb")
}
