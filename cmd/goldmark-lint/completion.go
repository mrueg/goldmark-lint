package main

import (
	"fmt"
	"io"
)

// writeCompletion writes a shell completion script for the given shell to w.
// Supported shells: bash, zsh, fish.
func writeCompletion(w io.Writer, shell string) error {
	switch shell {
	case "bash":
		fmt.Fprint(w, bashCompletion)
	case "zsh":
		fmt.Fprint(w, zshCompletion)
	case "fish":
		fmt.Fprint(w, fishCompletion)
	default:
		return fmt.Errorf("unsupported shell %q; supported shells: bash, zsh, fish", shell)
	}
	return nil
}

const bashCompletion = `# goldmark-lint bash completion script
# Source this file or add to ~/.bash_completion.d/ to enable tab completion.
_goldmark_lint_complete() {
    local cur prev words cword
    _init_completion || return

    local flags=(
        --config
        --completion
        --fail-on-warning
        --fix
        --fix-dry-run
        --format
        --help
        --list-rules
        --no-cache
        --no-globs
        --output-format
        --summary
        --version
        --watch
    )

    case "$prev" in
        --output-format)
            COMPREPLY=( $(compgen -W "default json junit tap sarif github" -- "$cur") )
            return 0
            ;;
        --config)
            COMPREPLY=( $(compgen -f -- "$cur") )
            return 0
            ;;
        --completion)
            COMPREPLY=( $(compgen -W "bash zsh fish" -- "$cur") )
            return 0
            ;;
    esac

    if [[ "$cur" == -* ]]; then
        COMPREPLY=( $(compgen -W "${flags[*]}" -- "$cur") )
        return 0
    fi

    COMPREPLY=( $(compgen -f -- "$cur") )
}

complete -F _goldmark_lint_complete goldmark-lint
`

const zshCompletion = `#compdef goldmark-lint
# goldmark-lint zsh completion script
# Place this file in a directory on your $fpath (e.g., ~/.zsh/completions/_goldmark-lint)
# and run: autoload -U compinit && compinit

_goldmark-lint() {
    local -a flags

    flags=(
        '(--config)--config[path to config file (overrides auto-discovery)]:config file:_files'
        '(--completion)--completion[generate shell completion script]:shell:(bash zsh fish)'
        '(--fail-on-warning)--fail-on-warning[exit with code 1 even when all violations are warnings]'
        '(--fix)--fix[updates files to resolve fixable issues]'
        '(--fix-dry-run)--fix-dry-run[show a diff of changes --fix would make, without modifying files]'
        '(--format)--format[read stdin, apply fixes, write stdout]'
        '(--help)--help[writes help message and exits]'
        '(--list-rules)--list-rules[print a table of all rules with their aliases, enabled/disabled state, and options]'
        '(--no-cache)--no-cache[disable reading/writing the cache file]'
        '(--no-globs)--no-globs[ignore the globs config key at runtime]'
        '(--output-format)--output-format[output format]:format:(default json junit tap sarif github)'
        '(--summary)--summary[print a count-per-rule breakdown after linting]'
        '(--version)--version[prints the version and exits]'
        '(--watch)--watch[re-lint files whenever they change (runs until Ctrl+C)]'
        '*:file:_files'
    )

    _arguments -s $flags
}

_goldmark-lint "$@"
`

const fishCompletion = `# goldmark-lint fish completion script
# Place this file at ~/.config/fish/completions/goldmark-lint.fish

# Disable file completions for flags that take specific values
complete -c goldmark-lint -n '__fish_contains_opt completion' -f

# --config: complete with file paths
complete -c goldmark-lint -l config -d 'path to config file (overrides auto-discovery)' -r -F

# --completion: complete with supported shells
complete -c goldmark-lint -l completion -d 'generate shell completion script' -r -f \
    -a 'bash\t"Bash completion script" zsh\t"Zsh completion script" fish\t"Fish completion script"'

# --output-format: complete with supported formats
complete -c goldmark-lint -l output-format -d 'output format' -r -f \
    -a 'default\t"Default human-readable format" json\t"JSON format" junit\t"JUnit XML format" tap\t"TAP format" sarif\t"SARIF format" github\t"GitHub Actions format"'

# Boolean flags
complete -c goldmark-lint -l fail-on-warning -d 'exit with code 1 even when all violations are warnings' -f
complete -c goldmark-lint -l fix -d 'updates files to resolve fixable issues' -f
complete -c goldmark-lint -l fix-dry-run -d 'show a diff of changes --fix would make, without modifying files' -f
complete -c goldmark-lint -l format -d 'read stdin, apply fixes, write stdout' -f
complete -c goldmark-lint -l help -d 'writes help message and exits' -f
complete -c goldmark-lint -l list-rules -d 'print a table of all rules' -f
complete -c goldmark-lint -l no-cache -d 'disable reading/writing the cache file' -f
complete -c goldmark-lint -l no-globs -d 'ignore the globs config key at runtime' -f
complete -c goldmark-lint -l summary -d 'print a count-per-rule breakdown after linting' -f
complete -c goldmark-lint -l version -d 'prints the version and exits' -f
complete -c goldmark-lint -l watch -d 're-lint files whenever they change' -f

# File completion for positional arguments (markdown files)
complete -c goldmark-lint -d 'markdown files to lint' -F
`
