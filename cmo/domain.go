package cmo

import (
	"context"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

// Domain is the cmo driver for the any-cli/kit framework.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "cmo",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "cmo",
			Short:  "Browse the Canadian Mathematical Olympiad archive from the command line.",
			Long: `cmo reads the Canadian Mathematical Society's CMO and CJMO archive at
cms.math.ca and returns structured records for every exam and solution PDF.
No API key required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/cmo-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "editions",
		Group:   "read",
		List:    true,
		Summary: "List all CMO and CJMO editions sorted newest-first",
		Args:    []kit.Arg{},
	}, listEditions)
}

// newClient builds the Client from kit-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClient()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.HTTP.Timeout = cfg.Timeout
	}
	return c, nil
}

// editionsInput is the input struct for the editions operation.
type editionsInput struct {
	Limit       int     `kit:"flag,inherit" help:"max editions to return (0 = all)"`
	Competition string  `kit:"flag"         help:"filter by competition: cmo or cjmo (empty = all)"`
	Client      *Client `kit:"inject"`
}

func listEditions(ctx context.Context, in editionsInput, emit func(*Edition) error) error {
	eds, err := in.Client.Editions(ctx, in.Limit)
	if err != nil {
		return err
	}
	comp := strings.ToUpper(strings.TrimSpace(in.Competition))
	for i := range eds {
		if comp != "" && eds[i].Competition != comp {
			continue
		}
		if err := emit(&eds[i]); err != nil {
			return err
		}
	}
	return nil
}

// Classify turns a URL or bare path into (type, id).
// For cmo we have a single resource type: "edition" identified by "YEAR:COMPETITION".
func (Domain) Classify(input string) (uriType, id string, err error) {
	// Accept direct URLs to the archive page.
	input = strings.TrimSpace(input)
	if strings.Contains(input, Host+"/competitions/cmo") {
		return "editions", "all", nil
	}
	return "", "", errs.Usage("unrecognized cmo reference: %q", input)
}

// Locate is the inverse of Classify: the https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	if uriType != "editions" {
		return "", errs.Usage("cmo has no resource type %q", uriType)
	}
	return BaseURL + archivePath, nil
}
