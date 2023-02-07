# Install autocomplete
#Install-Module PSReadLine

# install PSReadLine
#Install-Module -Name PSReadLine

# autocomplete
Import-Module PSReadLine
Set-PSReadLineOption -PredictionSource History

# ohmyposh
#oh-my-posh --init --shell pwsh --config ~/theme.omp.json | Invoke-Expression
oh-my-posh --init --shell pwsh --config ~/negligible.omp.json | Invoke-Expression
#oh-my-posh init pwsh --config "$env:POSH_THEMES_PATH/negligible.omp.json" | Invoke-Expression
Set-PSReadLineOption -PredictionSource History
Set-PSReadLineKeyHandler -Key "Ctrl+d" -Function MenuComplete
Set-PSReadLineKeyHandler -Key UpArrow -Function HistorySearchBackward
Set-PSReadLineKeyHandler -Key DownArrow -Function HistorySearchForward

# remove alias
Remove-Item alias:\diff -Force
## iex is the commandline(repl) of Elixir
Remove-Item alias:\iex -Force


# region conda initialize
# * installed by installer
# If (Test-Path "~\miniconda3\Scripts\conda.exe") {
#     (& "~\miniconda3\Scripts\conda.exe" "shell.powershell" "hook") | Out-String | ?{$_} | Invoke-Expression
# }

# * installed by scoop
If (Test-Path "~\scoop\apps\miniconda3\current\Scripts\conda.exe") {
    (& "~\scoop\apps\miniconda3\current\Scripts\conda.exe" "shell.powershell" "hook") | Out-String | Invoke-Expression
}
# not use base
# `conda config --set auto_activate_base false` is not compatible with ohmyposh
conda deactivate
#endregion


# jabba
if (Test-Path "~\.jabba\jabba.ps1") { . "~\.jabba\jabba.ps1" }