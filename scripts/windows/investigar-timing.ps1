<#
.SYNOPSIS
  Investiga como o UWW Timing se comunica com o LGTV (somente leitura).

.DESCRIPTION
  Rode no PC Windows com o UWW Timing e o LGTV ABERTOS e CONECTADOS.
  Etapa 1: processos, executaveis, portas abertas e conexoes do Timing/LGTV.
  Etapa 2: arquivos de configuracao nas pastas de instalacao (IP/porta/host).
  Etapa 3 (opcional, -Capturar): captura de pacotes com o pktmon nativo do
           Windows nas portas encontradas, enquanto voce aciona no Timing:
           challenge pedido/aceito/recusado, sangue, lesao e passividade.

  Nada e alterado no Timing nem no LGTV. Tudo vai para uma pasta na Area de Trabalho.

.EXAMPLE
  # Etapas 1 e 2 (nao precisa de administrador)
  powershell -ExecutionPolicy Bypass -File .\investigar-timing.ps1

.EXAMPLE
  # Etapas 1, 2 e 3 - PowerShell como Administrador, captura de 120 s
  powershell -ExecutionPolicy Bypass -File .\investigar-timing.ps1 -Capturar -Segundos 120
#>
param(
  [switch]$Capturar,
  [int]$Segundos = 120,
  [string]$Filtro = 'timing|lgtv|uww'
)

$ErrorActionPreference = 'Continue'
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$out = Join-Path ([Environment]::GetFolderPath('Desktop')) "cbw-investigacao-$stamp"
New-Item -ItemType Directory -Force -Path $out | Out-Null
$rel = Join-Path $out 'relatorio.txt'
function Log([string]$t) { $t | Tee-Object -FilePath $rel -Append }
function Sec([string]$t) { Log ''; Log ('=' * 70); Log $t; Log ('=' * 70) }

Sec "CBW Live - investigacao UWW Timing <=> LGTV  ($stamp)"
Log "Maquina: $env:COMPUTERNAME   Usuario: $env:USERNAME"
Log ("IPs: " + ((Get-NetIPAddress -AddressFamily IPv4 | Where-Object { $_.IPAddress -notlike '127.*' }).IPAddress -join ', '))

# ---------------- Etapa 1: processos e rede ----------------
Sec '1. Processos do Timing/LGTV'
$procs = Get-Process | Where-Object { $_.ProcessName -match $Filtro -or $_.MainWindowTitle -match $Filtro }
if (-not $procs) {
  Log "Nenhum processo encontrado com o filtro '$Filtro'. Abra o UWW Timing e o LGTV e rode de novo,"
  Log "ou rode com -Filtro 'parte-do-nome'. Processos com janela abertos agora:"
  Get-Process | Where-Object { $_.MainWindowTitle } | Sort-Object ProcessName |
    ForEach-Object { Log ("  {0,-30} {1}" -f $_.ProcessName, $_.MainWindowTitle) }
} else {
  foreach ($p in $procs) {
    $path = try { $p.Path } catch { '' }
    Log ("PID {0,-6} {1,-28} {2}" -f $p.Id, $p.ProcessName, $path)
    if ($p.MainWindowTitle) { Log ("            janela: {0}" -f $p.MainWindowTitle) }
  }
}
$pids = @($procs | ForEach-Object { $_.Id })
$nome = @{}; foreach ($p in $procs) { $nome[$p.Id] = $p.ProcessName }

Sec '2. Portas TCP (escutando e conexoes)'
$tcp = Get-NetTCPConnection -ErrorAction SilentlyContinue | Where-Object { $pids -contains $_.OwningProcess }
$tcp | Sort-Object State, LocalPort | ForEach-Object {
  Log ("{0,-12} {1,-8} local {2}:{3,-6} -> remoto {4}:{5,-6} [{6}]" -f $_.State, $nome[[int]$_.OwningProcess],
    $_.LocalAddress, $_.LocalPort, $_.RemoteAddress, $_.RemotePort, $_.OwningProcess)
}
if (-not $tcp) { Log '(nenhuma conexao TCP desses processos)' }

Sec '3. Portas UDP'
$udp = Get-NetUDPEndpoint -ErrorAction SilentlyContinue | Where-Object { $pids -contains $_.OwningProcess }
$udp | ForEach-Object { Log ("UDP {0,-8} {1}:{2} [{3}]" -f $nome[[int]$_.OwningProcess], $_.LocalAddress, $_.LocalPort, $_.OwningProcess) }
if (-not $udp) { Log '(nenhuma porta UDP desses processos)' }

Sec '4. Regras de firewall dos programas'
Get-NetFirewallApplicationFilter -ErrorAction SilentlyContinue |
  Where-Object { $_.Program -match $Filtro } |
  ForEach-Object {
    $r = $_ | Get-NetFirewallRule
    Log ("{0} | {1} | {2} | {3}" -f $r.DisplayName, $r.Direction, $r.Action, $_.Program)
  }

# ---------------- Etapa 2: configuracoes ----------------
Sec '5. Arquivos de configuracao (pastas de instalacao e AppData)'
$pastas = New-Object System.Collections.Generic.HashSet[string]
foreach ($p in $procs) { try { if ($p.Path) { [void]$pastas.Add((Split-Path $p.Path)) } } catch {} }
foreach ($base in @($env:APPDATA, $env:LOCALAPPDATA, $env:ProgramData, ${env:ProgramFiles}, ${env:ProgramFiles(x86)})) {
  if (-not $base) { continue }
  Get-ChildItem $base -Directory -ErrorAction SilentlyContinue |
    Where-Object { $_.Name -match $Filtro -or $_.Name -match 'united world' } |
    ForEach-Object { [void]$pastas.Add($_.FullName) }
}
$cfgDir = Join-Path $out 'configs'; New-Item -ItemType Directory -Force -Path $cfgDir | Out-Null
foreach ($d in $pastas) {
  Log "Pasta: $d"
  Get-ChildItem $d -Recurse -File -ErrorAction SilentlyContinue -Include *.json, *.ini, *.config, *.xml, *.cfg, *.yaml, *.yml, *.settings, *.properties |
    Where-Object { $_.Length -lt 2MB } |
    ForEach-Object {
      Log ("  {0}  ({1} bytes)" -f $_.FullName, $_.Length)
      $hits = Select-String -Path $_.FullName -Pattern 'host|port|ip|url|ws:|wss:|http|socket|udp|tcp|broadcast|multicast|server|remote|lgtv|timing' -ErrorAction SilentlyContinue | Select-Object -First 25
      foreach ($h in $hits) { Log ("      L{0}: {1}" -f $h.LineNumber, $h.Line.Trim()) }
      $dest = Join-Path $cfgDir (($_.FullName -replace '[:\\/]', '_'))
      Copy-Item $_.FullName $dest -ErrorAction SilentlyContinue
    }
}
Log ''
Log "ATENCAO: confira a pasta 'configs' antes de enviar - remova senhas, API keys e secrets se houver."

# ---------------- Etapa 3: captura ----------------
if ($Capturar) {
  Sec "6. Captura de pacotes ($Segundos s) com pktmon"
  $isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
  if (-not $isAdmin) {
    Log 'A captura precisa do PowerShell como Administrador. Etapas 1 e 2 concluidas.'
  } elseif (-not (Get-Command pktmon -ErrorAction SilentlyContinue)) {
    Log 'pktmon nao disponivel (Windows 10 1809+). Use o Wireshark manualmente nas portas acima.'
  } else {
    $portas = @($tcp | ForEach-Object { $_.LocalPort; $_.RemotePort }) + @($udp | ForEach-Object { $_.LocalPort }) |
      Where-Object { $_ -gt 0 -and $_ -ne 8080 } | Sort-Object -Unique
    pktmon stop 2>$null | Out-Null
    pktmon filter remove 2>$null | Out-Null
    if ($portas) {
      foreach ($pt in $portas | Select-Object -First 30) { pktmon filter add "p$pt" -p $pt | Out-Null }
      Log ("Portas filtradas: " + ($portas -join ', '))
    } else {
      Log 'Nenhuma porta especifica encontrada - capturando tudo exceto o Arena (arquivo maior).'
    }
    $etl = Join-Path $out 'captura.etl'
    pktmon start --capture --pkt-size 0 --file-name $etl | Out-Null
    Log ''
    Log ">>> CAPTURANDO por $Segundos s. Agora, no UWW Timing, faca NESTA ORDEM e anote o horario de cada um:"
    Log '    1. challenge pedido (vermelho)   2. challenge aceito/em analise   3. challenge recusado'
    Log '    4. tempo de sangue (start/stop)  5. tempo de lesao (start/stop)   6. passividade'
    Log '    7. um ponto qualquer (referencia)'
    for ($i = $Segundos; $i -gt 0; $i -= 10) { Write-Host "  ... $i s"; Start-Sleep -Seconds ([Math]::Min(10, $i)) }
    pktmon stop | Out-Null
    pktmon etl2pcap $etl --out (Join-Path $out 'captura.pcapng') | Out-Null
    pktmon filter remove | Out-Null
    Log "Captura salva: captura.pcapng (abre no Wireshark) e captura.etl"
  }
}

Sec 'Fim'
Log "Tudo em: $out"
Log 'Envie a pasta compactada (sem senhas/keys) junto com os horarios de cada acao.'
Start-Process explorer.exe $out
