@echo off
REM Integration test script for schoology-go on Windows
REM Requires 1Password CLI and .env.integration file

echo Checking for .env.integration file...
if not exist ".env.integration" (
    echo Error: .env.integration not found!
    echo.
    echo Please create .env.integration with your Schoology credentials.
    echo See docs/SESSION_EXTRACTION.md for instructions.
    exit /b 1
)

echo.
echo Running integration tests with 1Password...
echo NOTE: You will need to approve 1Password access
echo.

op run --env-file=.env.integration -- go test -tags=integration -v

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo Integration tests failed!
    echo Check your session credentials in .env.integration
    exit /b 1
)

echo.
echo Integration tests passed!
