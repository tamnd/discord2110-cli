package discord2110

import (
	"context"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

// Domain is the Discord archive driver.
type Domain struct{}

// Info describes the scheme, hostnames, and binary identity.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "discord2110",
		Hosts:  []string{Host, "discord.gg"},
		Identity: kit.Identity{
			Binary: "discord2110",
			Short:  "Archive and crawl Discord servers",
			Long: `discord2110 archives Discord servers via the Discord REST API v10.

The invite command is public (no token required).
The crawl command requires a DISCORD_TOKEN bot token.

Quick start:
  discord2110 invite ggTWnwK              server metadata from invite code
  discord2110 invite https://discord.gg/ggTWnwK
  DISCORD_TOKEN=your-bot-token discord2110 crawl 102860784329052160`,
			Site: Host,
			Repo: "https://github.com/tamnd/discord2110-cli",
		},
	}
}

// Register installs the client factory and all operations onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "invite",
		Group:   "public",
		Single:  true,
		Summary: "Resolve a Discord invite and show server metadata",
		Args:    []kit.Arg{{Name: "code", Help: "invite code or URL (e.g. ggTWnwK or discord.gg/ggTWnwK)"}},
	}, getInvite)

	kit.Handle(app, kit.OpMeta{
		Name:    "crawl",
		Group:   "archive",
		List:    true,
		Summary: "Crawl a Discord server and emit its channels and messages",
		Args:    []kit.Arg{{Name: "server-id", Help: "Discord server (guild) snowflake ID"}},
	}, crawlServer)
}

// newClient builds the Discord client from the kit-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	dcfg := DefaultConfig()
	if cfg.UserAgent != "" {
		dcfg.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		dcfg.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		dcfg.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		dcfg.Timeout = cfg.Timeout
	}
	return NewClient(dcfg), nil
}

// --- input structs ---

type inviteInput struct {
	Code   string  `kit:"arg"   help:"invite code or URL"`
	Client *Client `kit:"inject"`
}

type crawlInput struct {
	ServerID string  `kit:"arg"          help:"Discord server (guild) ID"`
	Token    string  `kit:"flag"         help:"Discord bot token (overrides DISCORD_TOKEN env var)"`
	Channels string  `kit:"flag"         help:"channel filter: text (default) or all"`
	Limit    int     `kit:"flag,inherit" help:"max messages per channel"`
	Client   *Client `kit:"inject"`
}

// --- handlers ---

// crawlRecord is a union type emitted by crawl: can be a server, channel, or message.
// We use Channel as the common emittable since it's a list op.
func getInvite(ctx context.Context, in inviteInput, emit func(*Server) error) error {
	srv, err := in.Client.GetInvite(ctx, in.Code)
	if err != nil {
		return mapErr(err)
	}
	return emit(srv)
}

func crawlServer(ctx context.Context, in crawlInput, emit func(*Channel) error) error {
	if in.Token != "" {
		in.Client.cfg.Token = in.Token
	}
	if in.Client.cfg.Token == "" {
		return errs.Usage("DISCORD_TOKEN is required for crawl; set the env var or use --token")
	}

	channels, err := in.Client.GetChannels(ctx, in.ServerID)
	if err != nil {
		return mapErr(err)
	}

	filterAll := in.Channels == "all"
	for _, ch := range channels {
		if !filterAll && !IsTextChannel(ch.Type) {
			continue
		}
		if err := emit(ch); err != nil {
			return err
		}
	}
	return nil
}

// Classify implements the URI resolver interface.
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("discord2110 reference is empty")
	}
	code := ExtractCode(input)
	if code != "" {
		return "invite", code, nil
	}
	return "server", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "invite":
		return "https://discord.gg/" + id, nil
	case "server":
		return "https://discord.com/channels/" + id, nil
	}
	return "", errs.Usage("discord2110 has no resource type %q", uriType)
}

// mapErr passes errors through; kit's typed error system handles exit codes.
func mapErr(err error) error {
	return err
}
