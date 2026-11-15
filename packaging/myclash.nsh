; Run before the installer copies files. A previous build can still hold the
; exe (and the single-instance lock) after its tray icon is gone. The first
; attempt is unelevated so normal upgrades are silent; if it is denied because
; an older instance is running as administrator, ask Windows to elevate just
; this taskkill call.
!include "LogicLib.nsh"

!macro customInit
  nsExec::ExecToStack 'taskkill /F /IM myclash.exe'
  Pop $0
  ${If} $0 != "0"
    ExecShell "runas" "$SYSDIR\taskkill.exe" "/F /IM myclash.exe"
    Sleep 2500
  ${EndIf}
!macroend
