$ErrorActionPreference = "Stop"

Write-Host "Generating protobuf code..." -ForegroundColor Green

# Создать директорию на корневом уровне (не внутри currency-service)
New-Item -ItemType Directory -Force -Path "pkg/grpc/pb" | Out-Null

# Сгенерировать код в общую папку
protoc `
  --proto_path=proto `
  --go_out=pkg/grpc/pb `
  --go_opt=paths=source_relative `
  --go-grpc_out=pkg/grpc/pb `
  --go-grpc_opt=paths=source_relative `
  proto/currency/currency.proto

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Protobuf code generated successfully!" -ForegroundColor Green
    Write-Host "  Generated files in: pkg/grpc/pb/" -ForegroundColor Cyan
} else {
    Write-Host "✗ Failed to generate protobuf code" -ForegroundColor Red
    exit 1
}
