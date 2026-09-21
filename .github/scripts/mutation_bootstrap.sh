#!/usr/bin/env bash
set -euo pipefail

# Trusted mutation bootstrap shared by GitHub Actions and Forgejo.
# The workflow copies this file from the base revision before checking out PR code.

mutation_script="${MUTATION_SCRIPT:-.ci/mutation.sh}"
runtime_parent="${RUNNER_TEMP:-/tmp}"
mkdir -p "$runtime_parent"
runtime_dir="$(mktemp -d "$runtime_parent/mutation-bootstrap.XXXXXX")"
execution_log="$runtime_dir/execution.log"
bootstrap_log="$runtime_dir/bootstrap.log"

cleanup() {
  rm -rf "$runtime_dir"
}
trap cleanup EXIT INT TERM

status() {
  printf 'mutation-bootstrap: %s\n' "$*"
}

fail_bootstrap() {
  printf 'ERROR[MUTATION_BOOTSTRAP]: %s\n' "$*" >&2
  exit 20
}

fail_engine() {
  printf 'ERROR[MUTATION_ENGINE]: %s\n' "$*" >&2
  exit 21
}

if [[ ! -f "$mutation_script" ]]; then
  fail_engine "$mutation_script is required for high-risk changes (${MUTATION_POLICY_LABELS:-unknown labels})."
fi

if grep -Eiq 'mutmut[[:space:]]+run' "$mutation_script"; then
  mutation_engine="mutmut"
elif grep -Eiq 'gremlins[[:space:]]+unleash' "$mutation_script"; then
  mutation_engine="gremlins"
elif grep -Eiq 'stryker([^[:alnum:]]+[^[:space:]]+)*[[:space:]]+run' "$mutation_script"; then
  mutation_engine="stryker"
elif grep -Eiq 'cargo[[:space:]]+mutants' "$mutation_script"; then
  mutation_engine="cargo-mutants"
elif grep -Eiq 'pitest|infection' "$mutation_script"; then
  mutation_engine="other"
else
  fail_engine "$mutation_script must invoke a supported real mutation engine."
fi

runtime_path="$PATH"
runtime_pythonpath="${PYTHONPATH:-}"
runtime_virtualenv=""
mutmut_spec='mutmut>=3,<4'

if [[ "$mutation_engine" == "mutmut" ]]; then
  python_bin="$(command -v python3 || true)"
  [[ -n "$python_bin" ]] || fail_bootstrap "python3 is unavailable; cannot bootstrap mutmut."

  venv_dir="$runtime_dir/venv"
  status "checking python3 -m venv support"
  if "$python_bin" -m venv "$venv_dir" >"$bootstrap_log" 2>&1 \
    && "$venv_dir/bin/python" -m pip --version >>"$bootstrap_log" 2>&1; then
    status "bootstrap mode=venv"
    if ! PIP_DISABLE_PIP_VERSION_CHECK=1 "$venv_dir/bin/python" -m pip install \
      --quiet --no-input "$mutmut_spec" >>"$bootstrap_log" 2>&1; then
      cat "$bootstrap_log" >&2
      fail_engine "mutmut installation failed inside the isolated virtual environment."
    fi
    runtime_path="$venv_dir/bin:$PATH"
    runtime_virtualenv="$venv_dir"
  else
    status "python venv unavailable; bootstrap mode=target"
    rm -rf "$venv_dir"
    target_dir="$runtime_dir/site"
    shim_dir="$runtime_dir/bin"
    mkdir -p "$target_dir" "$shim_dir"

    if "$python_bin" -m pip --version >>"$bootstrap_log" 2>&1; then
      if ! PIP_DISABLE_PIP_VERSION_CHECK=1 "$python_bin" -m pip install \
        --quiet --no-input --target "$target_dir" "$mutmut_spec" >>"$bootstrap_log" 2>&1; then
        cat "$bootstrap_log" >&2
        fail_engine "mutmut installation failed with pip --target."
      fi
    elif command -v uv >/dev/null 2>&1; then
      if ! uv pip install --quiet --target "$target_dir" "$mutmut_spec" >>"$bootstrap_log" 2>&1; then
        cat "$bootstrap_log" >&2
        fail_engine "mutmut installation failed with uv --target."
      fi
    else
      cat "$bootstrap_log" >&2 || true
      fail_bootstrap "venv/ensurepip is unavailable and neither python3 -m pip nor uv can provide the isolated target fallback."
    fi

    runtime_pythonpath="$target_dir${runtime_pythonpath:+:$runtime_pythonpath}"
    if [[ -x "$target_dir/bin/mutmut" ]]; then
      runtime_path="$target_dir/bin:$PATH"
    else
      cat >"$shim_dir/mutmut" <<EOF
#!/usr/bin/env bash
export PYTHONPATH="$target_dir${PYTHONPATH:+:$PYTHONPATH}"
exec "$python_bin" -c 'import importlib.metadata as m, sys; ep = next(ep for ep in m.entry_points(group="console_scripts") if ep.name == "mutmut"); sys.argv[0] = "mutmut"; raise SystemExit(ep.load()())' "\$@"
EOF
      chmod 0755 "$shim_dir/mutmut"
      runtime_path="$shim_dir:$PATH"
    fi
  fi

  if ! PATH="$runtime_path" PYTHONPATH="$runtime_pythonpath" command -v mutmut >/dev/null 2>&1; then
    fail_engine "mutmut is not installed after bootstrap."
  fi
  status "mutation engine=mutmut ready"
fi

set +e
PATH="$runtime_path" \
PYTHONPATH="$runtime_pythonpath" \
VIRTUAL_ENV="$runtime_virtualenv" \
bash "$mutation_script" 2>&1 | tee "$execution_log"
mutation_rc=${PIPESTATUS[0]}
set -e

if [[ "$mutation_rc" -eq 0 ]]; then
  status "result=success; mutation testing completed successfully"
  exit 0
fi

if grep -Eiq 'surviv(ed|ing)|suspicious|timed?[ -]?out|mutation gate failed' "$execution_log"; then
  printf 'ERROR[MUTATION_SURVIVORS]: mutation testing reported surviving or otherwise non-killed mutants.\n' >&2
else
  printf 'ERROR[MUTATION_EXECUTION]: mutation engine/test execution failed with exit code %s.\n' "$mutation_rc" >&2
fi
exit "$mutation_rc"
