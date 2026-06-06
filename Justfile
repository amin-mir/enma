refs_list := "references/repos.txt"

# List available recipes
default:
    @just --list

# Run all backend tests
test:
    cd backend && go test ./...

# Clone all reference repos listed in references/repos.txt (skips already-cloned)
refs-clone:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p references
    while IFS= read -r url || [[ -n "$url" ]]; do
        [[ -z "$url" || "$url" == \#* ]] && continue
        name=$(basename "$url")
        if [ ! -d "references/$name" ]; then
            echo "Cloning $url..."
            git clone "$url" "references/$name"
        else
            echo "references/$name already exists, skipping"
        fi
    done < {{refs_list}}

# Pull latest changes in all cloned reference repos
refs-pull:
    #!/usr/bin/env bash
    set -euo pipefail
    while IFS= read -r url || [[ -n "$url" ]]; do
        [[ -z "$url" || "$url" == \#* ]] && continue
        name=$(basename "$url")
        if [ -d "references/$name" ]; then
            echo "Pulling references/$name..."
            git -C "references/$name" pull --ff-only
        else
            echo "references/$name not cloned — run: just refs-clone"
        fi
    done < {{refs_list}}
