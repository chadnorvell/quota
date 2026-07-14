# quota

Terminal AI service quota dashboard for Codex and Claude subscriptions.

The project is built with Go, Bubble Tea, and Lip Gloss. The Nix flake provides
the Go toolchain and build dependencies.

## Run

```sh
nix develop
go run ./cmd/quota
```

For a non-interactive fetch:

```sh
go run ./cmd/quota once
```

Track additional accounts by passing their credentials directories. Each flag
is repeatable and the default `~/.codex` and `~/.claude` accounts remain
enabled:

```sh
quota --claude-dir ~/.claude-work --codex-dir ~/.codex-work
quota once --claude-dir ~/.claude-work
```

Or run the packaged binary:

```sh
nix run
```

## Consume As A Flake

Run directly from a checkout:

```sh
nix run path:/path/to/quota
```

Install into your user profile:

```sh
nix profile install path:/path/to/quota
quota
```

Use from another flake:

```nix
{
  inputs.quota.url = "github:chadnorvell/quota";

  outputs = { nixpkgs, quota, ... }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        packages = [
          quota.packages.${system}.default
        ];
      };
    };
}
```

If you keep this as a local/private checkout, replace the input URL with:

```nix
inputs.quota.url = "path:/path/to/quota";
```

Install globally through the NixOS module:

```nix
{
  inputs.quota.url = "github:chadnorvell/quota";

  outputs = { nixpkgs, quota, ... }: {
    nixosConfigurations.your-host = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        quota.nixosModules.default
        {
          programs.quota.enable = true;
        }
      ];
    };
  };
}
```

With the module imported elsewhere in your configuration, the usage is:

```nix
{
  imports = [ inputs.quota.nixosModules.default ];
  programs.quota.enable = true;
}
```

## Credentials

Codex:

- By default, the app reads `~/.codex/auth.json` and calls the Codex OAuth
  usage API used by CodexBar.
- `QUOTA_CODEX_COOKIE`, `OPENAI_COOKIE`, or `CHATGPT_COOKIE` enables the
  `https://chatgpt.com/codex/settings/usage` dashboard fallback.

Claude:

- By default, the app reads `~/.claude/.credentials.json` and calls Claude's
  OAuth usage endpoint.
- `ANTHROPIC_ADMIN_KEY` or `CLAUDE_ADMIN_KEY` enables Anthropic Admin API usage
  and cost report requests.
- `QUOTA_CLAUDE_COOKIE` or `CLAUDE_COOKIE` enables the Claude web API fallback.
  The value can be either a full `Cookie` header or a bare `sessionKey` value.

Extra accounts:

- Repeat `--codex-dir DIR` to read additional `DIR/auth.json` files.
- Repeat `--claude-dir DIR` to read additional `DIR/.credentials.json` files.
- `~` is expanded even when it is passed literally (for example,
  `--claude-dir=~/.claude-work`).

Keys:

- `u` or `tab`: toggle bars between used and remaining quota
- `r`: refresh immediately
- `q`, `esc`, or `ctrl-c`: quit
