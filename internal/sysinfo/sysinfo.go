// Package sysinfo 通过 PowerShell/CIM 采集整机硬件配置（页面1数据源）。
package sysinfo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"whatismypc/internal/psexec"
)

// psScript 一次性采集全部配置信息，输出压缩 JSON。
// 使用 CIM 类（与语言无关），适配中英文系统。
const psScript = `
$ErrorActionPreference = 'SilentlyContinue'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$cs    = Get-CimInstance Win32_ComputerSystem
$os    = Get-CimInstance Win32_OperatingSystem
$cpu   = Get-CimInstance Win32_Processor | Select-Object -First 1
$board = Get-CimInstance Win32_BaseBoard | Select-Object -First 1
$bios  = Get-CimInstance Win32_BIOS | Select-Object -First 1
$mem = @(Get-CimInstance Win32_PhysicalMemory | ForEach-Object {
  [pscustomobject]@{
    Bank = $_.BankLabel; DeviceLocator = $_.DeviceLocator
    CapacityGB = [math]::Round($_.Capacity/1GB,1); SpeedMHz = $_.Speed
    ConfiguredMHz = $_.ConfiguredClockSpeed; Manufacturer = $_.Manufacturer
    PartNumber = ("$($_.PartNumber)").Trim()
  }
})
$gpus = @(Get-CimInstance Win32_VideoController | ForEach-Object {
  $res = ""
  if ($_.CurrentHorizontalResolution -gt 0) { $res = "$($_.CurrentHorizontalResolution)x$($_.CurrentVerticalResolution)" }
  [pscustomobject]@{
    Name = $_.Name; VRAMGB = [math]::Round($_.AdapterRAM/1GB,1)
    DriverVersion = $_.DriverVersion; Manufacturer = $_.AdapterCompatibility
    Resolution = $res
  }
})
$disks = @(Get-CimInstance Win32_DiskDrive | Sort-Object Index | ForEach-Object {
  [pscustomobject]@{
    Model = $_.Model; InterfaceType = $_.InterfaceType; MediaType = $_.MediaType
    SizeGB = [math]::Round($_.Size/1GB,1); Firmware = $_.FirmwareRevision
    Serial = ("$($_.SerialNumber)").Trim()
  }
})
$volumes = @(Get-CimInstance Win32_LogicalDisk -Filter "DriveType=3" | ForEach-Object {
  [pscustomobject]@{
    Drive = $_.DeviceID; Label = $_.VolumeName; FileSystem = $_.FileSystem
    SizeGB = [math]::Round($_.Size/1GB,1); FreeGB = [math]::Round($_.FreeSpace/1GB,1)
  }
})
$nets = @(Get-CimInstance Win32_NetworkAdapter -Filter "PhysicalAdapter='True'" | ForEach-Object {
  [pscustomobject]@{
    Name = $_.Name; Manufacturer = $_.Manufacturer
    SpeedMbps = [math]::Round($_.Speed/1e6,0); MAC = $_.MACAddress
  }
})
$ips = @((Get-CimInstance Win32_NetworkAdapterConfiguration -Filter "IPEnabled='True'" |
  ForEach-Object { $_.IPAddress } | Where-Object { $_ -match '^\d+\.\d+\.\d+\.\d+$' }) | Select-Object -Unique)
$monitors = @()
try {
  $monitors = @(Get-CimInstance -Namespace root/wmi WmiMonitorID | ForEach-Object {
    [pscustomobject]@{
      Name = ((($_.UserFriendlyName | Where-Object { $_ -ne 0 }) | ForEach-Object { [char]$_ }) -join '')
      Manufacturer = ((($_.ManufacturerName | Where-Object { $_ -ne 0 }) | ForEach-Object { [char]$_ }) -join '')
      Year = $_.YearOfManufacture; Week = $_.WeekOfManufacture
    }
  })
} catch {}
$result = [pscustomobject]@{
  computerName = "$($cs.Name)"
  model = "$($cs.Manufacturer) $($cs.Model)"
  os = [pscustomobject]@{ Caption = $os.Caption; Version = $os.Version; Build = $os.BuildNumber; Arch = $os.OSArchitecture }
  cpu = [pscustomobject]@{
    Name = "$($cpu.Name)".Trim(); Manufacturer = $cpu.Manufacturer
    Cores = $cpu.NumberOfCores; Threads = $cpu.NumberOfLogicalProcessors
    MaxClockMHz = $cpu.MaxClockSpeed; L2KB = $cpu.L2CacheSize; L3KB = $cpu.L3CacheSize
    Socket = $cpu.SocketDesignation
  }
  board  = [pscustomobject]@{ Manufacturer = $board.Manufacturer; Product = $board.Product }
  bios   = [pscustomobject]@{ Manufacturer = $bios.Manufacturer; Version = $bios.SMBIOSBIOSVersion; ReleaseDate = "$($bios.ReleaseDate)" }
  memory = [pscustomobject]@{ TotalGB = [math]::Round($cs.TotalPhysicalMemory/1GB,1); Modules = $mem }
  gpus = $gpus; disks = $disks; volumes = $volumes
  networkAdapters = $nets; ipAddresses = $ips; monitors = $monitors
}
$result | ConvertTo-Json -Depth 6 -Compress
`

// Collect 执行采集，返回解析后的 map（保留原始结构给前端）。
func Collect() (map[string]interface{}, error) {
	cmd := psexec.Command("-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", psScript)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("采集失败: %v %s", err, errb.String())
	}
	b := bytes.TrimSpace(out.Bytes())
	if len(b) == 0 || b[0] != '{' {
		return nil, fmt.Errorf("采集结果无效: %s", string(b))
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("解析失败: %v", err)
	}
	return m, nil
}

// CacheDuration 前端缓存时长建议（配置信息基本不变）。
const CacheDuration = 30 * time.Second
