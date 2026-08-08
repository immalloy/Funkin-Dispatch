$statePath = Join-Path $PSScriptRoot '..\state.json'
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText($statePath, '{}', $utf8NoBom)
Write-Host "state.json reset" -ForegroundColor Green
