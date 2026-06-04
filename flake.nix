{
  description = "Terminal AI service quota dashboard";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        src = pkgs.lib.cleanSourceWith {
          src = ./.;
          filter = path: type:
            let
              name = builtins.baseNameOf path;
            in
            name != ".go" && name != "result" && name != ".direnv";
        };
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "quota";
          version = "0.1.0";
          inherit src;
          vendorHash = "sha256-vj6i7Uc5LXnOF3Gi/GKy+FQ/I6eSyt2kKgZl8C5u2MM=";
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            go-tools
            golangci-lint
          ];

          shellHook = ''
            export GOPATH="$PWD/.go"
            export GOMODCACHE="$PWD/.go/pkg/mod"
            export PATH="$GOPATH/bin:$PATH"
          '';
        };

        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.default;
        };
      });
}
