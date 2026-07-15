$bin = if ($env:GOBIN) { $env:GOBIN } elseif ($env:GOPATH) { Join-Path $env:GOPATH 'bin' } else { Join-Path $HOME 'go\bin' }
New-Item -ItemType Directory -Force $bin | Out-Null
go build -o (Join-Path $bin 'devpkgs.exe') .
Write-Host "Installed devpkgs to $bin\devpkgs.exe"
