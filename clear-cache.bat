@echo off
REM Clear Redis cache for blogs and press releases
REM This is needed after updating the data structure to include author details

echo Clearing Redis cache for blogs and press releases...

REM Load .env file
for /f "tokens=1,2 delims==" %%a in (.env) do (
    if "%%a"=="REDIS_HOST" set REDIS_HOST=%%b
    if "%%a"=="REDIS_PORT" set REDIS_PORT=%%b
    if "%%a"=="REDIS_PASSWORD" set REDIS_PASSWORD=%%b
)

REM Clear cache patterns using redis-cli
redis-cli -h %REDIS_HOST% -p %REDIS_PORT% -a %REDIS_PASSWORD% --no-auth-warning --scan --pattern "blog:*" | redis-cli -h %REDIS_HOST% -p %REDIS_PORT% -a %REDIS_PASSWORD% --no-auth-warning --pipe del
redis-cli -h %REDIS_HOST% -p %REDIS_PORT% -a %REDIS_PASSWORD% --no-auth-warning --scan --pattern "blogs:*" | redis-cli -h %REDIS_HOST% -p %REDIS_PORT% -a %REDIS_PASSWORD% --no-auth-warning --pipe del
redis-cli -h %REDIS_HOST% -p %REDIS_PORT% -a %REDIS_PASSWORD% --no-auth-warning --scan --pattern "press_release:*" | redis-cli -h %REDIS_HOST% -p %REDIS_PORT% -a %REDIS_PASSWORD% --no-auth-warning --pipe del
redis-cli -h %REDIS_HOST% -p %REDIS_PORT% -a %REDIS_PASSWORD% --no-auth-warning --scan --pattern "press_releases:*" | redis-cli -h %REDIS_HOST% -p %REDIS_PORT% -a %REDIS_PASSWORD% --no-auth-warning --pipe del

echo Cache cleared successfully!
pause
