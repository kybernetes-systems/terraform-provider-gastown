.PHONY: test testacc clean

test:
	go test ./...

testacc:
	@echo "Running acceptance tests with signal trapping..."
	@trap 'echo "Cleaning up..."; pkill -f "dolt sql-server" || true; pkill -f "beads dolt idle-monitor" || true' EXIT SIGINT SIGTERM; \
	TF_ACC=1 go test ./... -v

clean:
	@echo "Cleaning up orphan processes..."
	pkill -f "dolt sql-server" || true
	pkill -f "beads dolt idle-monitor" || true
	rm -rf /tmp/TestAcc_*
	rm -rf /tmp/TestHQResource_*
