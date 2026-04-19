@echo off
REM Test script for schoology-go on Windows

echo Running unit tests...
go test -v
if %ERRORLEVEL% NEQ 0 (
    echo Unit tests failed!
    exit /b 1
)

echo.
echo Running tests with coverage...
go test -cover -coverprofile=coverage.out
if %ERRORLEVEL% NEQ 0 (
    echo Coverage tests failed!
    exit /b 1
)

echo.
echo Generating coverage report...
go tool cover -html=coverage.out -o coverage.html
echo Coverage report generated: coverage.html

echo.
echo Running go fmt...
go fmt ./...

echo.
echo Running go vet...
go vet ./...
if %ERRORLEVEL% NEQ 0 (
    echo go vet found issues!
    exit /b 1
)

echo.
echo All tests passed!
echo.
echo To run integration tests with 1Password:
echo   op run --env-file=.env.integration -- go test -tags=integration -v
