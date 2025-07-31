# API Tester Server Runner
$Host.UI.RawUI.WindowTitle = "API Tester Server"

Write-Host "🚀 Starting Go API Tester Server..." -ForegroundColor Green
Write-Host ""

try {
  go run main.go
}
catch {
  Write-Host "❌ Error starting server: $_" -ForegroundColor Red
}
finally {
  Write-Host ""
  Write-Host "Server stopped. Press any key to close this window..." -ForegroundColor Yellow
  $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
}