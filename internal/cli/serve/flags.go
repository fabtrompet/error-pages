package serve

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"github.com/tarampampam/error-pages/internal/env"
)

type flags struct {
	listen struct {
		ip   string
		port uint16
	}
	template struct {
		name string
	}
	defaultErrorPage string
	defaultHTTPCode  uint16
	showDetails      bool
	proxyHTTPHeaders string // comma-separated
}

// HeadersToProxy converts a comma-separated string with headers list into strings slice (with a sorting and without
// duplicates).
func (f *flags) HeadersToProxy() []string {
	var raw = strings.Split(f.proxyHTTPHeaders, ",")

	if len(raw) == 0 {
		return []string{}
	} else if len(raw) == 1 {
		if h := strings.TrimSpace(raw[0]); h != "" {
			return []string{h}
		} else {
			return []string{}
		}
	}

	var m = make(map[string]struct{}, len(raw))

	// make unique and ignore empty strings
	for _, h := range raw {
		if h = strings.TrimSpace(h); h != "" {
			if _, ok := m[h]; !ok {
				m[h] = struct{}{}
			}
		}
	}

	// convert map into slice
	var headers = make([]string, 0, len(m))
	for h := range m {
		headers = append(headers, h)
	}

	// make sort
	sort.Strings(headers)

	return headers
}

const (
	listenFlagName           = "listen"
	portFlagName             = "port"
	templateNameFlagName     = "template-name"
	defaultErrorPageFlagName = "default-error-page"
	defaultHTTPCodeFlagName  = "default-http-code"
	showDetailsFlagName      = "show-details"
	proxyHTTPHeadersFlagName = "proxy-headers"
)

const (
	useRandomTemplate              = "random"
	useRandomTemplateOnEachRequest = "i-said-random"
	useRandomTemplateDaily         = "random-daily"
	useRandomTemplateHourly        = "random-hourly"
)

func (f *flags) init(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(
		&f.listen.ip,
		listenFlagName, "l",
		"0.0.0.0",
		fmt.Sprintf("IP address to listen on [$%s]", env.ListenAddr),
	)
	flagSet.Uint16VarP(
		&f.listen.port,
		portFlagName, "p",
		8080, //nolint:gomnd // must be same as default healthcheck `--port` flag value
		fmt.Sprintf("TCP port number [$%s]", env.ListenPort),
	)
	flagSet.StringVarP(
		&f.template.name,
		templateNameFlagName, "t",
		"",
		fmt.Sprintf(
			"template name (set \"%s\" to use a randomized or \"%s\" to use a randomized template on each request "+
				"or \"%s/%s\" daily/hourly randomized) [$%s]",
			useRandomTemplate,
			useRandomTemplateOnEachRequest,
			useRandomTemplateDaily,
			useRandomTemplateHourly,
			env.TemplateName,
		),
	)
	flagSet.StringVarP(
		&f.defaultErrorPage,
		defaultErrorPageFlagName, "",
		"404",
		fmt.Sprintf("default error page [$%s]", env.DefaultErrorPage),
	)
	flagSet.Uint16VarP(
		&f.defaultHTTPCode,
		defaultHTTPCodeFlagName, "",
		404, //nolint:gomnd
		fmt.Sprintf("default HTTP response code [$%s]", env.DefaultHTTPCode),
	)
	flagSet.BoolVarP(
		&f.showDetails,
		showDetailsFlagName, "",
		false,
		fmt.Sprintf("show request details in response [$%s]", env.ShowDetails),
	)
	flagSet.StringVarP(
		&f.proxyHTTPHeaders,
		proxyHTTPHeadersFlagName, "",
		"",
		fmt.Sprintf("proxy HTTP request headers list (comma-separated) [$%s]", env.ProxyHTTPHeaders),
	)
}

func (f *flags) overrideUsingEnv(flagSet *pflag.FlagSet) (lastErr error) { //nolint:gocognit,gocyclo
	flagSet.VisitAll(func(flag *pflag.Flag) {
		// flag was NOT defined using CLI (flags should have maximal priority)
		if !flag.Changed { //nolint:nestif
			switch flag.Name {
			case listenFlagName:
				if envVar, exists := env.ListenAddr.Lookup(); exists {
					f.listen.ip = strings.TrimSpace(envVar)
				}

			case portFlagName:
				if envVar, exists := env.ListenPort.Lookup(); exists {
					if p, err := strconv.ParseUint(envVar, 10, 16); err == nil { //nolint:gomnd
						f.listen.port = uint16(p)
					} else {
						lastErr = fmt.Errorf("wrong TCP port environment variable [%s] value", envVar)
					}
				}

			case templateNameFlagName:
				if envVar, exists := env.TemplateName.Lookup(); exists {
					f.template.name = strings.TrimSpace(envVar)
				}

			case defaultErrorPageFlagName:
				if envVar, exists := env.DefaultErrorPage.Lookup(); exists {
					f.defaultErrorPage = strings.TrimSpace(envVar)
				}

			case defaultHTTPCodeFlagName:
				if envVar, exists := env.DefaultHTTPCode.Lookup(); exists {
					if code, err := strconv.ParseUint(envVar, 10, 16); err == nil { //nolint:gomnd
						f.defaultHTTPCode = uint16(code)
					} else {
						lastErr = fmt.Errorf("wrong default HTTP response code environment variable [%s] value", envVar)
					}
				}

			case showDetailsFlagName:
				if envVar, exists := env.ShowDetails.Lookup(); exists {
					if b, err := strconv.ParseBool(envVar); err == nil {
						f.showDetails = b
					}
				}

			case proxyHTTPHeadersFlagName:
				if envVar, exists := env.ProxyHTTPHeaders.Lookup(); exists {
					f.proxyHTTPHeaders = strings.TrimSpace(envVar)
				}
			}
		}
	})

	return lastErr
}

func (f *flags) validate() error {
	if net.ParseIP(f.listen.ip) == nil {
		return fmt.Errorf("wrong IP address [%s] for listening", f.listen.ip)
	}

	if f.defaultHTTPCode > 599 { //nolint:gomnd
		return fmt.Errorf("wrong default HTTP response code [%d]", f.defaultHTTPCode)
	}

	if strings.ContainsRune(f.proxyHTTPHeaders, ' ') {
		return fmt.Errorf("whitespaces in the HTTP headers for proxying [%s] are not allowed", f.proxyHTTPHeaders)
	}

	return nil
}