'{}' | Out-File -Encoding utf8 -NoNewline -LiteralPath "$PSScriptRoot\..\state.json"
Write-Host "state.json reset" -ForegroundColor Green
