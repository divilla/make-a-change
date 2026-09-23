FLOW_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GIT_ROOT := $(shell git rev-parse --show-toplevel 2>/dev/null)
GIT_BRANCH := $(shell git branch --show-current 2>/dev/null)
CHANGE_SLUG := $(patsubst change/%,%,$(GIT_BRANCH))
TEMP_DIR := $(GIT_ROOT)/$${MCH_TEMP_DIR}/$${MCH_REF_UUID}/$${MCH_STAGE}
SPEC_TEMP_DIR := $(GIT_ROOT)/$${MCH_TEMP_DIR}/$${MCH_REF_UUID}/spec
SPEC_REVIEW_TEMP_DIR := $(GIT_ROOT)/$${MCH_TEMP_DIR}/$${MCH_REF_UUID}/spec-review
DEFS_DIR := $(GIT_ROOT)/agent/defs
SPECS_DIR := $(GIT_ROOT)/specs
FLOW_STEPS := def-write def-review artifact-chat spec-write pr-write spec-review spec-review-chat
FLOW_ENTRY_TARGETS := $(addsuffix -entry,$(FLOW_STEPS))
FLOW_EXEC_TARGETS := $(addsuffix -exec,$(FLOW_STEPS))
FLOW_EXIT_TARGETS := $(addsuffix -exit,$(FLOW_STEPS))
FLOW_BEFORE_TARGETS := $(addsuffix -before,$(FLOW_STEPS))
FLOW_RUN_TARGETS := $(addsuffix -run,$(FLOW_STEPS))
FLOW_LEAVE_TARGETS := $(addsuffix -leave,$(FLOW_STEPS))

.PHONY: help list init diff def-prepare def-init def-write def-review artifact-chat
.PHONY: spec-prepare spec-init spec-write pr-write spec-review spec-review-chat verify-exit fix-exit commit extract-session
.PHONY: $(FLOW_ENTRY_TARGETS) $(FLOW_EXEC_TARGETS) $(FLOW_EXIT_TARGETS)
.PHONY: $(FLOW_BEFORE_TARGETS) $(FLOW_RUN_TARGETS) $(FLOW_LEAVE_TARGETS)

def-% spec-% pr-% artifact-%: export MCH_STAGE := artifact
spec-review spec-review-%: export MCH_STAGE := spec-review
code-% polish-%: export MCH_STAGE := code
review-% fix-%: export MCH_STAGE := code-review
merge-% stage-% master-%: export MCH_STAGE := merge

help: list

list:
	@printf '%s\n' 'Default mch Flow targets:'
	@printf '%s\n' '  <stage>-entry   run the stage entry hook'
	@printf '%s\n' '  <stage>-exec    run the stage exec hook'
	@printf '%s\n' '  <stage>-exit    run the stage exit hook'
	@printf '%s\n' '  <stage>-before  alias for <stage>-entry'
	@printf '%s\n' '  <stage>-run     alias for <stage>-exec'
	@printf '%s\n' '  <stage>-leave   alias for <stage>-exit'

init:
	@if [ -z "$(GIT_ROOT)" ]; then printf '%s\n' 'cannot extract repo root' >&2; exit 1; fi
	@if [ -z "$$MCH_TEMP_DIR" ]; then printf '%s\n' 'missing MCH_TEMP_DIR' >&2; exit 1; fi
	@if [ -z "$$MCH_REF_UUID" ]; then printf '%s\n' 'missing MCH_REF_UUID' >&2; exit 1; fi
	@if [ -z "$$MCH_STAGE" ]; then printf '%s\n' 'missing MCH_STAGE' >&2; exit 1; fi
	@case "$$MCH_TEMP_DIR" in /*|*..*) printf 'invalid MCH_TEMP_DIR: %s\n' "$$MCH_TEMP_DIR" >&2; exit 1;; esac
	@printf '%s\n' "$$MCH_REF_UUID" | grep -Eq '^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$$' || { printf 'invalid MCH_REF_UUID: %s\n' "$$MCH_REF_UUID" >&2; exit 1; }
	@mkdir -p "$(TEMP_DIR)"
	@mkdir -p "$(SPEC_TEMP_DIR)"
	@mkdir -p "$(SPEC_REVIEW_TEMP_DIR)"

def-prepare: init
	@cp "$(DEFS_DIR)/$(CHANGE_SLUG).md" "$(TEMP_DIR)/output.md";

spec-prepare: init
	@cp "$(SPECS_DIR)/$(CHANGE_SLUG).md" "$(SPEC_TEMP_DIR)/output.md";

def-init: init
	@cp "$(TEMP_DIR)/output.md" "$(TEMP_DIR)/input.md";

spec-init: init
	@cp "$(SPEC_TEMP_DIR)/output.md" "$(SPEC_TEMP_DIR)/input.md";

diff:
	@git diff --no-index --color=never --unified=1 -- "$(TEMP_DIR)/input.md" "$(TEMP_DIR)/output.md" | bat --language=diff -pp --theme 'Coldark-Dark' --style=plain --color=always --paging=never

def-write: def-prepare def-init
	@$(FLOW_DIR)scripts/codex-exec-restore-session.sh "$(FLOW_DIR)prompts/def-write.md"

def-review: def-init
	@$(FLOW_DIR)scripts/codex-exec-restore-session.sh "$(FLOW_DIR)prompts/def-review.md"

spec-write: def-init
	@$(FLOW_DIR)scripts/codex-exec-restore-session.sh "$(FLOW_DIR)prompts/spec-write.md"

pr-write: def-init
	@$(FLOW_DIR)scripts/codex-exec-restore-session.sh "$(FLOW_DIR)prompts/pr-write.md"

artifact-chat: init
	@$(FLOW_DIR)scripts/codex-resume-session.sh

spec-review: init
	@cp "$(SPECS_DIR)/$(CHANGE_SLUG).md" "$(SPEC_REVIEW_TEMP_DIR)/output.md"
	@cp "$(SPEC_REVIEW_TEMP_DIR)/output.md" "$(SPEC_REVIEW_TEMP_DIR)/input.md"
	@$(FLOW_DIR)scripts/codex-exec-new-session.sh \
		"$(FLOW_DIR)prompts/spec-review.md"

spec-review-chat: init
	@$(FLOW_DIR)scripts/codex-resume-session.sh

verify-exit: init
	@git add -A
	@git commit -m "Spec fix for $(CHANGE_SLUG) by agent" || true
	@git push origin "change/$(CHANGE_SLUG)"

fix-exit: init
	@git add -A
	@git commit -m "Code fix for $(CHANGE_SLUG) by agent" || true
	@git push origin "change/$(CHANGE_SLUG)"

commit:
	@git add -A
	@git commit -m "Update for $(CHANGE_SLUG) by user"
	@git push origin "change/$(CHANGE_SLUG)"


extract-session: init
	@repo="$$(git rev-parse --show-toplevel)"; \
	stderr_file="$$(mktemp)"; \
	events="$$(codex exec -C "$$repo" --json "$$prompt" 2>"$$stderr_file")"; \
	status="$$?"; \
	stderr="$$(cat "$$stderr_file")"; \
	rm -f "$$stderr_file"; \
	session_id="$$(printf '%s\n' "$$events" | jq -rsr '\
	  [ \
		.[] \
		| select(.type=="thread.started") \
		| (.thread_id // .session_id // .session.id // .id // empty) \
	  ] \
	  | first // "" \
	')"; \
	final_output="$$(printf '%s\n' "$$events" | jq -rsr '\
	  [ \
		.[] \
		| if .type=="item.completed" and .item.type=="agent_message" then \
			.item.text \
		  elif .type=="agent_message" or .type=="message" then \
			(.message // .text // .output // .content // empty) \
		  else \
			empty \
		  end \
	  ] \
	  | last // "" \
	')"; \
	printf 'status=%s\n' "$$status"; \
	printf 'session_id=%s\n' "$$session_id"; \
	printf 'stderr=%s\n' "$$stderr"; \
	printf 'final_output=%s\n' "$$final_output"; \
	if [ "$$status" -ne 0 ]; then \
	  exit "$$status"; \
	fi; \
	if [ "$$final_output" != "No questions or suggestions." ]; then \
	  if [ -z "$$session_id" ]; then \
		printf '%s\n' 'missing session_id; cannot resume' >&2; \
		exit 1; \
	  fi; \
	  codex -C "$$repo" resume "$$session_id"; \
	fi

$(FLOW_ENTRY_TARGETS):
	@$(FLOW_DIR)scripts/demo-entry.sh "$(patsubst %-entry,%,$@)"

$(FLOW_EXEC_TARGETS):
	@$(FLOW_DIR)scripts/demo-exec.sh "$(patsubst %-exec,%,$@)"

$(FLOW_EXIT_TARGETS):
	@$(FLOW_DIR)scripts/demo-exit.sh "$(patsubst %-exit,%,$@)"

$(FLOW_BEFORE_TARGETS):
	@$(MAKE) --no-print-directory -f $(lastword $(MAKEFILE_LIST)) "$(patsubst %-before,%-entry,$@)"

$(FLOW_RUN_TARGETS):
	@$(MAKE) --no-print-directory -f $(lastword $(MAKEFILE_LIST)) "$(patsubst %-run,%-exec,$@)"

$(FLOW_LEAVE_TARGETS):
	@$(MAKE) --no-print-directory -f $(lastword $(MAKEFILE_LIST)) "$(patsubst %-leave,%-exit,$@)"
