# demos/Makefile — Record demo GIFs from VHS tape files.
#
# Usage (from the demos/ directory):
#   make profiles   — record the profiles demo (interactive, needs human)
#   make sessions   — record the sessions demo (fully automated)
#   make clean      — remove generated GIFs
#
# Prerequisites:
#   - vhs (brew install vhs)
#   - claude-profile binary already built (run `make build` from repo root)

# If there's a local claude-profile binary, prefer that!
export PATH := ../bin:$(PATH)
export CLAUDE_PROFILES_DIR=/tmp/claude-profile

.PHONY: profiles sessions clean

setup:
	$(MAKE) -C .. build

# Record the profiles demo (outputs docs/demo.gif).
# Interactive — requires human for OAuth step.
profiles: clean setup
	vhs profiles.tape

# Record the sessions demo (outputs docs/sessions-demo.gif).
# Fully automated.
sessions: setup
	#bash setup-demo-data.sh
	vhs sessions.tape

# Remove generated GIFs.
clean:
	rm -rf "$(CLAUDE_PROFILES_DIR)"
	rm -f ../docs/demo.gif ../docs/sessions-demo.gif
