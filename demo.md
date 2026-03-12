# Terraform Provider Gas Town tests

*2026-03-12T18:09:03Z by Showboat 0.6.1*
<!-- showboat-id: 0858368f-8c20-4e31-bace-c4eceeda8e0a -->

Here is the result of running unit tests and acceptance tests on the Gas Town Terraform provider with the Mock gt: unknown command  execution correctly mocked for test environments.

```bash
export PATH="/home/linuxbrew/.linuxbrew/bin:$PATH" && make test
```

```output
go test ./...
?   	github.com/kybernetes-systems/terraform-provider-gastown	[no test files]
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/exec	0.030s
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/gastown/crew	(cached)
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/gastown/hq	0.485s
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/gastown/rig	(cached)
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/provider	0.184s
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/testutil	(cached)
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/validators	(cached)
```

```bash
export PATH="/home/linuxbrew/.linuxbrew/bin:$PATH" && make testacc
```

```output
Running acceptance tests with signal trapping...
?   	github.com/kybernetes-systems/terraform-provider-gastown	[no test files]
=== RUN   TestNotFoundError_Error
=== RUN   TestNotFoundError_Error/with_resource_and_name
=== RUN   TestNotFoundError_Error/with_resource_only
=== RUN   TestNotFoundError_Error/empty_error
--- PASS: TestNotFoundError_Error (0.00s)
    --- PASS: TestNotFoundError_Error/with_resource_and_name (0.00s)
    --- PASS: TestNotFoundError_Error/with_resource_only (0.00s)
    --- PASS: TestNotFoundError_Error/empty_error (0.00s)
=== RUN   TestIsNotFound
=== RUN   TestIsNotFound/nil_error
=== RUN   TestIsNotFound/direct_NotFoundError
=== RUN   TestIsNotFound/wrapped_NotFoundError
=== RUN   TestIsNotFound/regular_error
=== RUN   TestIsNotFound/string_containing_not_found
--- PASS: TestIsNotFound (0.00s)
    --- PASS: TestIsNotFound/nil_error (0.00s)
    --- PASS: TestIsNotFound/direct_NotFoundError (0.00s)
    --- PASS: TestIsNotFound/wrapped_NotFoundError (0.00s)
    --- PASS: TestIsNotFound/regular_error (0.00s)
    --- PASS: TestIsNotFound/string_containing_not_found (0.00s)
=== RUN   TestIsNotFound_ErrorAs
--- PASS: TestIsNotFound_ErrorAs (0.00s)
=== RUN   TestRunner_GT_version
--- PASS: TestRunner_GT_version (0.00s)
=== RUN   TestRunner_GT_nonzeroExitReturnsError
--- PASS: TestRunner_GT_nonzeroExitReturnsError (0.00s)
PASS
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/exec	0.014s
=== RUN   TestCrewResource_Create_callsCrewAdd
--- PASS: TestCrewResource_Create_callsCrewAdd (0.00s)
=== RUN   TestCrewResource_Read_findsCrew
--- PASS: TestCrewResource_Read_findsCrew (0.00s)
=== RUN   TestCrewResource_Read_removesIfNotFound
--- PASS: TestCrewResource_Read_removesIfNotFound (0.00s)
=== RUN   TestCrewResource_Delete_callsCrewRemove
--- PASS: TestCrewResource_Delete_callsCrewRemove (0.00s)
=== RUN   TestCrewResource_Update_fails
--- PASS: TestCrewResource_Update_fails (0.00s)
PASS
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/gastown/crew	(cached)
=== RUN   TestHQResource_Create_callsGtInstall
--- PASS: TestHQResource_Create_callsGtInstall (0.09s)
=== RUN   TestHQResource_Read_idempotent
--- PASS: TestHQResource_Read_idempotent (0.05s)
=== RUN   TestHQResource_ForceNew_onPathChange
--- PASS: TestHQResource_ForceNew_onPathChange (0.00s)
=== RUN   TestHQResource_Delete_callsUninstall
--- PASS: TestHQResource_Delete_callsUninstall (0.05s)
PASS
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/gastown/hq	0.204s
=== RUN   TestRigResource_Injection
    injection_test.go:37: Validator correctly caught malicious name: [{{Name must contain only alphanumeric characters, hyphens, and underscores: myrig --extra-flag Invalid Name} {[name]}}]
--- PASS: TestRigResource_Injection (0.00s)
=== RUN   TestRigResource_Create_callsRigAdd
--- PASS: TestRigResource_Create_callsRigAdd (0.00s)
=== RUN   TestRigResource_Create_setsRuntime
--- PASS: TestRigResource_Create_setsRuntime (0.00s)
=== RUN   TestRigResource_Delete_stopsAndDocks
--- PASS: TestRigResource_Delete_stopsAndDocks (0.00s)
=== RUN   TestRigResource_Update_runtime
--- PASS: TestRigResource_Update_runtime (0.00s)
PASS
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/gastown/rig	(cached)
=== RUN   TestAcc_FullLifecycle
--- PASS: TestAcc_FullLifecycle (4.84s)
=== RUN   TestAcc_DriftScenario
    acceptance_test.go:50: requires real gt CLI: rig Read does not parse runtime from filesystem, so config.json edits do not cause detectable drift via FakeRunner
--- SKIP: TestAcc_DriftScenario (0.00s)
=== RUN   TestAcc_Concurrency
=== PAUSE TestAcc_Concurrency
=== RUN   TestProvider_schema_hq_path
--- PASS: TestProvider_schema_hq_path (0.00s)
=== RUN   TestProvider_configure_rejects_empty_hq_path
--- PASS: TestProvider_configure_rejects_empty_hq_path (0.00s)
=== RUN   TestProvider_registers_three_resources
--- PASS: TestProvider_registers_three_resources (0.00s)
=== RUN   TestHQ_Schema_NoBeads
--- PASS: TestHQ_Schema_NoBeads (0.00s)
=== RUN   TestProvider_NewForTesting
--- PASS: TestProvider_NewForTesting (0.00s)
=== CONT  TestAcc_Concurrency
=== RUN   TestAcc_Concurrency/first
=== PAUSE TestAcc_Concurrency/first
=== RUN   TestAcc_Concurrency/second
=== PAUSE TestAcc_Concurrency/second
=== CONT  TestAcc_Concurrency/first
=== CONT  TestAcc_Concurrency/second
--- PASS: TestAcc_Concurrency (0.05s)
    --- PASS: TestAcc_Concurrency/first (3.06s)
    --- PASS: TestAcc_Concurrency/second (3.06s)
PASS
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/provider	7.972s
=== RUN   TestFakeRunner_HQPath
--- PASS: TestFakeRunner_HQPath (0.00s)
=== RUN   TestFakeRunner_Install_CreatesFilesystem
--- PASS: TestFakeRunner_Install_CreatesFilesystem (0.00s)
=== RUN   TestFakeRunner_RigLifecycle
--- PASS: TestFakeRunner_RigLifecycle (0.00s)
=== RUN   TestFakeRunner_CrewLifecycle
--- PASS: TestFakeRunner_CrewLifecycle (0.00s)
=== RUN   TestFakeRunner_BD_Status
--- PASS: TestFakeRunner_BD_Status (0.00s)
PASS
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/testutil	(cached)
=== RUN   TestPathValidator
=== RUN   TestPathValidator/valid_absolute_path
=== RUN   TestPathValidator/relative_path
=== RUN   TestPathValidator/parent_directory
=== RUN   TestPathValidator/shell_injection
=== RUN   TestPathValidator/backticks
=== RUN   TestPathValidator/variable_injection
--- PASS: TestPathValidator (0.00s)
    --- PASS: TestPathValidator/valid_absolute_path (0.00s)
    --- PASS: TestPathValidator/relative_path (0.00s)
    --- PASS: TestPathValidator/parent_directory (0.00s)
    --- PASS: TestPathValidator/shell_injection (0.00s)
    --- PASS: TestPathValidator/backticks (0.00s)
    --- PASS: TestPathValidator/variable_injection (0.00s)
=== RUN   TestSafeNameValidator
=== RUN   TestSafeNameValidator/valid_name
=== RUN   TestSafeNameValidator/with_spaces
=== RUN   TestSafeNameValidator/with_dots
=== RUN   TestSafeNameValidator/with_slash
=== RUN   TestSafeNameValidator/flag_injection
=== RUN   TestSafeNameValidator/command_injection
--- PASS: TestSafeNameValidator (0.00s)
    --- PASS: TestSafeNameValidator/valid_name (0.00s)
    --- PASS: TestSafeNameValidator/with_spaces (0.00s)
    --- PASS: TestSafeNameValidator/with_dots (0.00s)
    --- PASS: TestSafeNameValidator/with_slash (0.00s)
    --- PASS: TestSafeNameValidator/flag_injection (0.00s)
    --- PASS: TestSafeNameValidator/command_injection (0.00s)
=== RUN   TestRepoURLValidator
=== RUN   TestRepoURLValidator/valid_https
=== RUN   TestRepoURLValidator/valid_ssh
=== RUN   TestRepoURLValidator/valid_local
=== RUN   TestRepoURLValidator/shell_injection
=== RUN   TestRepoURLValidator/newlines
--- PASS: TestRepoURLValidator (0.00s)
    --- PASS: TestRepoURLValidator/valid_https (0.00s)
    --- PASS: TestRepoURLValidator/valid_ssh (0.00s)
    --- PASS: TestRepoURLValidator/valid_local (0.00s)
    --- PASS: TestRepoURLValidator/shell_injection (0.00s)
    --- PASS: TestRepoURLValidator/newlines (0.00s)
PASS
ok  	github.com/kybernetes-systems/terraform-provider-gastown/internal/validators	(cached)
Cleaning up...
```
