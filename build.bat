@echo off
setlocal enabledelayedexpansion

set PROJECT_ROOT=%~dp0
set FRONTEND_DIR=%PROJECT_ROOT%frontend

goto :parse_args

:print_header
echo.
echo ==================================
echo %~1
echo ==================================
echo.
exit /b

:print_success
echo [92m%~1[0m
exit /b

:print_error
echo [91m%~1[0m
exit /b

:build_dev_tools
call :print_header "Building Developer Tools"
cd /d "%PROJECT_ROOT%"

if not exist "%PROJECT_ROOT%bin" mkdir "%PROJECT_ROOT%bin"

echo Building plugin-validator...
go build -o bin\plugin-validator.exe .\cmd\plugin-validator
if errorlevel 1 (
    call :print_error "Failed to build plugin-validator"
    exit /b 1
)
call :print_success "Built plugin-validator"

echo Building plugin-doc-generator...
go build -o bin\plugin-doc-generator.exe .\cmd\plugin-doc-generator
if errorlevel 1 (
    call :print_error "Failed to build plugin-doc-generator"
    exit /b 1
)
call :print_success "Built plugin-doc-generator"

echo Building plugin-doc-generator-all...
go build -o bin\plugin-doc-generator-all.exe .\cmd\plugin-doc-generator-all
if errorlevel 1 (
    call :print_error "Failed to build plugin-doc-generator-all"
    exit /b 1
)
call :print_success "Built plugin-doc-generator-all"

echo Building plugin-scaffolder...
go build -o bin\plugin-scaffolder.exe .\cmd\plugin-scaffolder
if errorlevel 1 (
    call :print_error "Failed to build plugin-scaffolder"
    exit /b 1
)
call :print_success "Built plugin-scaffolder"

call :print_success "All developer tools built successfully"
exit /b 0

:build_frontend
call :print_header "Building Frontend"
cd /d "%FRONTEND_DIR%"

call npm run build
if errorlevel 1 (
    call :print_error "Frontend build failed"
    exit /b 1
)

call :print_success "Frontend build completed"
exit /b 0

:copy_resources
call :print_header "Copying Resources"
cd /d "%PROJECT_ROOT%"

if not exist "%PROJECT_ROOT%build\bin" (
    call :print_error "Build bin directory not found"
    exit /b 1
)

set HAS_ERROR=0

if exist "%PROJECT_ROOT%examples" (
    xcopy /E /I /Y "%PROJECT_ROOT%examples" "%PROJECT_ROOT%build\bin\examples" > nul
    if errorlevel 1 (
        call :print_error "Failed to copy examples"
        set HAS_ERROR=1
    ) else (
        call :print_success "Examples copied successfully"
    )
) else (
    call :print_error "Examples directory not found"
    set HAS_ERROR=1
)

if exist "%PROJECT_ROOT%scripts" (
    xcopy /E /I /Y "%PROJECT_ROOT%scripts" "%PROJECT_ROOT%build\bin\scripts" > nul
    if errorlevel 1 (
        call :print_error "Failed to copy scripts"
        set HAS_ERROR=1
    ) else (
        call :print_success "Scripts copied successfully"
    )
) else (
    call :print_error "Scripts directory not found"
    set HAS_ERROR=1
)

if exist "%PROJECT_ROOT%plugins" (
    xcopy /E /I /Y "%PROJECT_ROOT%plugins" "%PROJECT_ROOT%build\bin\plugins" > nul
    if errorlevel 1 (
        call :print_error "Failed to copy plugins"
        set HAS_ERROR=1
    ) else (
        call :print_success "Plugins copied successfully"
    )
) else (
    call :print_error "Plugins directory not found"
    set HAS_ERROR=1
)

if exist "%PROJECT_ROOT%bin" (
    if not exist "%PROJECT_ROOT%build\bin\tools" mkdir "%PROJECT_ROOT%build\bin\tools"
    xcopy /E /I /Y "%PROJECT_ROOT%bin\*" "%PROJECT_ROOT%build\bin\tools\" > nul
    if errorlevel 1 (
        call :print_error "Failed to copy developer tools"
        set HAS_ERROR=1
    ) else (
        call :print_success "Developer tools copied successfully"
    )
) else (
    call :print_error "Warning: Developer tools not found (run 'build.bat tools' first)"
)

exit /b !HAS_ERROR!

:build_wails
call :print_header "Building Wails Application"
cd /d "%PROJECT_ROOT%"

if "%~1"=="" (
    set PLATFORM=windows/amd64
) else (
    set PLATFORM=%~1
)

wails build -platform !PLATFORM!
if errorlevel 1 (
    call :print_error "Wails build failed"
    exit /b 1
)

call :copy_resources
if errorlevel 1 (
    call :print_error "Warning: Failed to copy some resources, but build succeeded"
)

call :print_success "Wails build completed for !PLATFORM!"
echo.
echo Executable location: %PROJECT_ROOT%build\bin\
dir /b "%PROJECT_ROOT%build\bin\"
exit /b 0

:clean_build
call :print_header "Cleaning Build Artifacts"
cd /d "%PROJECT_ROOT%"

if exist "%FRONTEND_DIR%\dist" rmdir /s /q "%FRONTEND_DIR%\dist"
if exist "%PROJECT_ROOT%build" rmdir /s /q "%PROJECT_ROOT%build"

call :print_success "Build artifacts cleaned"
exit /b 0

:show_help
echo Cauldron Build Script (Windows)
echo.
echo Usage: build.bat [COMMAND] [OPTIONS]
echo.
echo Commands:
echo   frontend         Build only the frontend
echo   wails [PLATFORM] Build the Wails application (default: windows/amd64)
echo   tools            Build developer tools (plugin-validator, etc.)
echo   all [PLATFORM]   Build tools, frontend, and Wails app (default)
echo   clean            Clean all build artifacts
echo   rebuild [PLATFORM] Clean and rebuild everything
echo   help             Show this help message
echo.
echo Platforms:
echo   windows/amd64    Windows 64-bit (default)
echo   linux/amd64      Linux 64-bit (requires cross-compilation)
echo.
echo Examples:
echo   build.bat                    Build everything for Windows
echo   build.bat frontend           Build only frontend
echo   build.bat tools              Build developer tools
echo   build.bat wails              Build Wails app for Windows
echo   build.bat rebuild            Clean and rebuild everything
echo   build.bat clean              Clean build artifacts
echo.
exit /b 0

:parse_args
if "%~1"=="" set COMMAND=all
if not "%~1"=="" set COMMAND=%~1

if /i "%COMMAND%"=="frontend" (
    call :build_frontend
    goto :end
)

if /i "%COMMAND%"=="tools" (
    call :build_dev_tools
    goto :end
)

if /i "%COMMAND%"=="wails" (
    call :build_wails %~2
    goto :end
)

if /i "%COMMAND%"=="all" (
    call :build_dev_tools
    if errorlevel 1 goto :end
    call :build_frontend
    if errorlevel 1 goto :end
    call :build_wails %~2
    if errorlevel 1 goto :end
    call :print_header "Build Complete!"
    call :print_success "Application ready to run"
    goto :end
)

if /i "%COMMAND%"=="clean" (
    call :clean_build
    goto :end
)

if /i "%COMMAND%"=="rebuild" (
    call :clean_build
    call :build_dev_tools
    if errorlevel 1 goto :end
    call :build_frontend
    if errorlevel 1 goto :end
    call :build_wails %~2
    if errorlevel 1 goto :end
    call :print_header "Rebuild Complete!"
    call :print_success "Application ready to run"
    goto :end
)

if /i "%COMMAND%"=="help" (
    call :show_help
    goto :end
)

if /i "%COMMAND%"=="--help" (
    call :show_help
    goto :end
)

if /i "%COMMAND%"=="-h" (
    call :show_help
    goto :end
)

call :print_error "Unknown command: %COMMAND%"
echo.
call :show_help
exit /b 1

:end
endlocal
