// Package monitor 采集实时占用率、温度与磁盘 S.M.A.R.T. 状态（页面2数据源）。
//
// 为缩短单次采样耗时，采集分为两条路径：
//   - SnapshotFast：高频指标（CPU/内存/GPU 占用、温度、卷占用、磁盘活动）
//   - SnapshotSlow：静态/慢速信息（物理磁盘清单、S.M.A.R.T. 可靠性计数器）
//
// 慢速部分由 bridge 层缓存并低频刷新。全部进程静默后台执行，不弹窗。
package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"

	"whatismypc/internal/psexec"
)

const psFastScript = `
$ErrorActionPreference = 'SilentlyContinue'
$ProgressPreference = 'SilentlyContinue'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$cpuPct = (Get-CimInstance Win32_PerfFormattedData_PerfOS_Processor -Filter "Name='_Total'" | Select-Object -First 1).PercentProcessorTime
$os = Get-CimInstance Win32_OperatingSystem
$memTotalGB = [math]::Round($os.TotalVisibleMemorySize/1MB,1)
$memUsedGB  = [math]::Round(($os.TotalVisibleMemorySize - $os.FreePhysicalMemory)/1MB,1)
$gpuPct = (Get-CimInstance Win32_PerfFormattedData_GPUPerformanceCounters_GPUEngine |
  Measure-Object -Property UtilizationPercentage -Maximum).Maximum
$zoneTemps = @()
try {
  $zoneTemps = @(Get-CimInstance -Namespace root/wmi MSAcpi_ThermalZoneTemperature |
    ForEach-Object { [math]::Round(($_.CurrentTemperature - 2732)/10, 1) })
} catch {}
$gpuTemp = $null
if (Get-Command nvidia-smi -ErrorAction SilentlyContinue) {
  $t = (& nvidia-smi --query-gpu=temperature.gpu --format=csv,noheader 2>$null | Select-Object -First 1)
  if ("$t" -match '^\d+') { $gpuTemp = [int]$t }
}
$volumes = @(Get-CimInstance Win32_LogicalDisk -Filter "DriveType=3" | ForEach-Object {
  [pscustomobject]@{
    Drive = $_.DeviceID; Label = $_.VolumeName
    SizeGB = [math]::Round($_.Size/1GB,1); FreeGB = [math]::Round($_.FreeSpace/1GB,1)
  }
})
$diskActivity = @{}
Get-CimInstance Win32_PerfFormattedData_PerfDisk_PhysicalDisk -ErrorAction SilentlyContinue |
  Where-Object { $_.Name -ne '_Total' } | ForEach-Object {
    $idx = ("$($_.Name)" -split ' ')[0]
    if (-not $diskActivity.ContainsKey($idx)) {
      $diskActivity[$idx] = [pscustomobject]@{ Pct = 0; MBps = 0.0 }
    }
    $diskActivity[$idx].Pct = [math]::Min(100, [math]::Max($diskActivity[$idx].Pct, $_.PercentDiskTime))
    $diskActivity[$idx].MBps = [math]::Round($diskActivity[$idx].MBps + $_.DiskBytesPersec/1MB, 1)
  }
[pscustomobject]@{
  cpuPct = $cpuPct
  memTotalGB = $memTotalGB; memUsedGB = $memUsedGB
  gpuPct = $gpuPct
  zoneTemps = $zoneTemps; gpuTemp = $gpuTemp
  volumes = $volumes
  diskActivity = $diskActivity
} | ConvertTo-Json -Depth 6 -Compress
`

const psSlowScript = `
$ErrorActionPreference = 'SilentlyContinue'
$ProgressPreference = 'SilentlyContinue'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$physical = @(Get-PhysicalDisk | Sort-Object DeviceId | ForEach-Object {
  $rel = $_ | Get-StorageReliabilityCounter -ErrorAction SilentlyContinue
  $tempC = $null
  if ($rel -and $rel.Temperature -gt 0) {
    $tempC = $rel.Temperature
    if ($tempC -gt 200) { $tempC = [math]::Round($tempC - 273.15, 1) }
  }
  [pscustomobject]@{
    Id = $_.DeviceId; Name = $_.FriendlyName; MediaType = "$($_.MediaType)"
    Bus = "$($_.BusType)"; Health = "$($_.HealthStatus)"; OpStatus = "$($_.OperationalStatus)"
    SizeGB = [math]::Round($_.Size/1GB,1); TempC = $tempC
    Wear = $rel.Wear; PowerOnHours = $rel.PowerOnHours; StartStopCycles = $rel.StartStopCycleCount
  }
})
$smart = @()
$smartIdx = 0
Get-CimInstance -Namespace root/wmi MSStorageDriver_FailurePredictStatus -ErrorAction SilentlyContinue | ForEach-Object {
  $smart += [pscustomobject]@{ Index = $smartIdx; Instance = $_.InstanceName; PredictFailure = $_.PredictFailure; Reason = $_.Reason }
  $smartIdx++
}
[pscustomobject]@{
  physicalDisks = $physical
  smartPredict = $smart
} | ConvertTo-Json -Depth 6 -Compress
`

func runPS(script string) (map[string]interface{}, error) {
	cmd := psexec.Command("-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("采样失败: %v %s", err, errb.String())
	}
	b := bytes.TrimSpace(out.Bytes())
	if len(b) == 0 || b[0] != '{' {
		return nil, fmt.Errorf("采样结果无效: %s", string(b))
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("解析失败: %v", err)
	}
	return m, nil
}

// SnapshotFast 高频采样（每次轮询都调用）。
func SnapshotFast() (map[string]interface{}, error) {
	return runPS(psFastScript)
}

// SnapshotSlow 低频采样（磁盘清单与 S.M.A.R.T.，耗时较长，由上层缓存）。
func SnapshotSlow() (map[string]interface{}, error) {
	return runPS(psSlowScript)
}
