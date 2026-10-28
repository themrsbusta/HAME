param([string]$ConfigPath = (Join-Path $PSScriptRoot 'config.json'), [switch]$Managed, [switch]$SelfTest, [string]$EnginePath = "")
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
[Windows.Forms.Application]::EnableVisualStyles()
$ConfigPath = [IO.Path]::GetFullPath($ConfigPath)
$base = Split-Path -Parent $ConfigPath
$exe = Join-Path $PSScriptRoot 'HAME.exe'
if($EnginePath){$exe=$EnginePath}
$form = New-Object Windows.Forms.Form
$form.Text = 'HAME 0.9 RC - Configurador'
$form.ClientSize = New-Object Drawing.Size(850,690)
$form.StartPosition = 'CenterScreen'
$form.FormBorderStyle = 'FixedDialog'
$form.MaximizeBox = $false
$form.Font = New-Object Drawing.Font('Segoe UI',10)
if (Test-Path (Join-Path $PSScriptRoot 'HAME.ico')) { $form.Icon = New-Object Drawing.Icon((Join-Path $PSScriptRoot 'HAME.ico')) }
function Label($text,$x,$y,$width=200) { $c=New-Object Windows.Forms.Label; $c.Text=$text; $c.SetBounds($x,$y,$width,25); $form.Controls.Add($c); return $c }
function TextBox($x,$y,$width=560) { $c=New-Object Windows.Forms.TextBox; $c.SetBounds($x,$y,$width,28); $form.Controls.Add($c); return $c }
function Button($text,$x,$y,$width=110) { $c=New-Object Windows.Forms.Button; $c.Text=$text; $c.SetBounds($x,$y,$width,32); $form.Controls.Add($c); return $c }
function Combo($x,$y,$width) { $c=New-Object Windows.Forms.ComboBox; $c.DropDownStyle='DropDownList'; $c.SetBounds($x,$y,$width,30); $form.Controls.Add($c); return $c }
$null=Label 'Motor de inferência' 20 20
$backend=Combo 215 18 200; $null=$backend.Items.Add('llama-server'); $null=$backend.Items.Add('llama-cli'); $backend.SelectedIndex=0
$description=Label 'Server: HTTP e contexto pré-carregado. CLI: processo por tarefa; recarrega na próxima mensagem.' 20 53 810
$null=Label 'Executável do llama' 20 98
$llama=TextBox 215 95 480; $browseLlama=Button 'Procurar...' 710 94 115
$detect=Button 'Detectar GPUs / versão' 215 133 220
$null=Label 'Modelo principal (GGUF)' 20 188
$main=TextBox 215 185 480; $browseMain=Button 'Procurar...' 710 184 115
$null=Label 'Modelo compilador (GGUF)' 20 229
$compiler=TextBox 215 226 480; $browseCompiler=Button 'Procurar...' 710 225 115
$null=Label 'GPU do principal' 20 272
$gpuMain=Combo 215 269 610
$null=Label 'GPU do compilador' 20 313
$gpuCompiler=Combo 215 310 610
$null=Label 'Pasta de contexto' 20 356
$data=TextBox 215 353 480; $browseData=Button 'Procurar...' 710 352 115
$null=Label 'Modo de execução' 20 397
$mode=Combo 215 394 180; $null=$mode.Items.Add('constrained'); $null=$mode.Items.Add('resident'); $mode.SelectedIndex=0
$null=Label 'Porta do chat' 440 397 120
$port=New-Object Windows.Forms.NumericUpDown; $port.SetBounds(590,394,120,28); $port.Minimum=1024; $port.Maximum=65535; $port.Value=8088; $form.Controls.Add($port)
$null=Label 'Contexto (tokens)' 20 440
$ctx=New-Object Windows.Forms.NumericUpDown; $ctx.SetBounds(215,437,120,28); $ctx.Minimum=2048; $ctx.Maximum=1048576; $ctx.Increment=1024; $ctx.Value=8192; $form.Controls.Add($ctx)
$null=Label 'Saída máxima' 370 440 120
$output=New-Object Windows.Forms.NumericUpDown; $output.SetBounds(500,437,100,28); $output.Minimum=128; $output.Maximum=65536; $output.Value=1024; $form.Controls.Add($output)
$null=Label 'Camadas GPU' 620 440 110
$layers=TextBox 735 437 90; $layers.Text='auto'
$notice=Label 'CPU usa o backend de CPU. Vulkan/CUDA/etc. aparecem somente se o llama escolhido os detectar.' 20 482 810
$status=New-Object Windows.Forms.TextBox; $status.Multiline=$true; $status.ReadOnly=$true; $status.ScrollBars='Vertical'; $status.SetBounds(20,516,805,95); $form.Controls.Add($status)
$test=Button 'Testar' 20 631 120
$save=Button 'Salvar' 435 631 120
$start=Button 'Salvar e iniciar' 570 631 155
$cancel=Button 'Fechar' 740 631 85
$script:DeviceRows=@(); $script:Profile=$null; $script:Task=$null; $script:AfterTask=''; $script:SaveAction=''; $script:Pending=$null; $script:CheckPath=$null; $script:SavedAndStart=$false
function Device-Choices($devices) { $script:DeviceRows=@(@{id='auto';name='';text='Automático (decisão do llama)'},@{id='none';name='';text='Somente CPU (sem offload para GPU)'})
 foreach($d in $devices){$script:DeviceRows+=@{id=$d.id;name=$d.name;text=($d.id+' — '+$d.name+' — '+$d.memory)}}
 foreach($box in @($gpuMain,$gpuCompiler)) { $box.Items.Clear(); foreach($row in $script:DeviceRows){$null=$box.Items.Add($row.text)}; $box.SelectedIndex=0 }
}
function Selected-Device($box){return $script:DeviceRows[$box.SelectedIndex]}
function Select-Device($box,$id){for($i=0;$i -lt $script:DeviceRows.Count;$i++){if($script:DeviceRows[$i].id -eq $id){$box.SelectedIndex=$i;return}}}
function Resolve-ConfigPath($text){if([IO.Path]::IsPathRooted($text)){return [IO.Path]::GetFullPath($text)};return [IO.Path]::GetFullPath((Join-Path $base $text))}
function Portable-Path($text){$full=Resolve-ConfigPath $text; $prefix=$base.TrimEnd('\')+'\';if($full.StartsWith($prefix,[StringComparison]::OrdinalIgnoreCase)){return $full.Substring($prefix.Length)};return $full}
Device-Choices @()
$data.Text='data'
$script:Original=@{}
if(Test-Path -LiteralPath $ConfigPath){try{$old=Get-Content -LiteralPath $ConfigPath -Raw|ConvertFrom-Json;foreach($prop in $old.PSObject.Properties){$script:Original[$prop.Name]=$prop.Value};if($old.backend -eq 'cli'){$backend.SelectedIndex=1;$llama.Text=$old.llama_cli}else{$llama.Text=$old.llama_server};$main.Text=$old.main_model;$compiler.Text=$old.compiler_model;if($old.data_dir){$data.Text=$old.data_dir};if($old.mode -eq 'resident'){$mode.SelectedIndex=1};if($old.context_tokens){$ctx.Value=$old.context_tokens};if($old.output_tokens){$output.Value=$old.output_tokens};if($old.gpu_layers){$layers.Text=$old.gpu_layers};if($old.listen){$port.Value=[int]($old.listen.Split(':')[-1])};$status.Text='Configuração carregada. Clique em Detectar GPUs para confirmar o executável e os dispositivos.'}catch{$status.Text='Não foi possível ler a configuração anterior: '+$_}}else{$status.Text='Primeira execução: selecione o llama e os dois GGUF, detecte as GPUs e salve.'}
$chooseFile={param($box,$filter);$dialog=New-Object Windows.Forms.OpenFileDialog;$dialog.Filter=$filter;if($dialog.ShowDialog($form)-eq 'OK'){$box.Text=$dialog.FileName};$dialog.Dispose()}
$browseLlama.Add_Click({&$chooseFile $llama 'Executável llama (*.exe)|*.exe'})
$browseMain.Add_Click({&$chooseFile $main 'Modelos GGUF (*.gguf)|*.gguf'})
$browseCompiler.Add_Click({&$chooseFile $compiler 'Modelos GGUF (*.gguf)|*.gguf'})
$browseData.Add_Click({$dialog=New-Object Windows.Forms.FolderBrowserDialog;if($dialog.ShowDialog($form)-eq 'OK'){$data.Text=$dialog.SelectedPath};$dialog.Dispose()})
$backend.Add_SelectedIndexChanged({$script:Profile=$null;Device-Choices @();if($llama.Text){$sibling=Join-Path (Split-Path -Parent $llama.Text) ('llama-'+@('server','cli')[$backend.SelectedIndex]+'.exe');if(Test-Path -LiteralPath $sibling){$llama.Text=$sibling}};if($backend.SelectedIndex -eq 1){$mode.SelectedIndex=0;$mode.Enabled=$false}else{$mode.Enabled=$true};$status.Text='Motor alterado. Detecte novamente as GPUs e a versão.'})
$llama.Add_TextChanged({$script:Profile=$null})
function Set-Busy($busy){foreach($c in @($backend,$llama,$detect,$browseLlama,$main,$compiler,$browseMain,$browseCompiler,$gpuMain,$gpuCompiler,$data,$browseData,$mode,$port,$ctx,$output,$layers,$test,$save,$start)){$c.Enabled=-not $busy};if($backend.SelectedIndex -eq 1){$mode.Enabled=$false}}
function Begin-Task($arguments,$kind){if($script:Task){return};$info=New-Object Diagnostics.ProcessStartInfo;$info.FileName=$exe;$info.Arguments=$arguments;$info.WorkingDirectory=$base;$info.UseShellExecute=$false;$info.CreateNoWindow=$true;$info.RedirectStandardOutput=$true;$info.RedirectStandardError=$true;$info.StandardOutputEncoding=[Text.Encoding]::UTF8;$info.StandardErrorEncoding=[Text.Encoding]::UTF8;$process=New-Object Diagnostics.Process;$process.StartInfo=$info;$null=$process.Start();$script:ReadOut=$process.StandardOutput.ReadToEndAsync();$script:ReadErr=$process.StandardError.ReadToEndAsync();$script:Task=$process;$script:AfterTask=$kind;Set-Busy $true;$timer.Start()}
$timer=New-Object Windows.Forms.Timer;$timer.Interval=200
$timer.Add_Tick({if(-not $script:Task -or -not $script:Task.HasExited){return};$timer.Stop();$process=$script:Task;$text=$script:ReadOut.Result;$errors=$script:ReadErr.Result;$code=$process.ExitCode;$script:Task=$null;$process.Dispose();Set-Busy $false;try{
 if($code -ne 0){throw ($text+"`r`n"+$errors)}
 if($script:AfterTask -eq 'probe'){$p=$text|ConvertFrom-Json;if($p.error){throw $p.error};$script:Profile=$p;Device-Choices $p.devices;Select-Device $gpuMain $script:Original.main_device;Select-Device $gpuCompiler $script:Original.compiler_device;$status.Text=$p.version+"`r`n"+([string]$p.devices.Count)+' GPU(s) detectada(s). CPU permanece disponível. Alterações de versão são verificadas por capacidades.'}
 elseif($script:AfterTask -eq 'check'){$status.Text='Configuração válida. '+$text;if($script:SaveAction -ne 'test'){$json=$script:Pending|ConvertTo-Json -Depth 8;[IO.File]::WriteAllText($ConfigPath,$json,(New-Object Text.UTF8Encoding($false)));$status.Text+=' Salva em '+$ConfigPath;if($script:SaveAction -eq 'start'){$script:SavedAndStart=$true;$form.Close();if(-not $Managed){Start-Process -FilePath $exe -ArgumentList ('-config "'+$ConfigPath+'" -open') -WorkingDirectory $base}}}}
 }catch{$status.Text='Falha: '+$_}finally{if($script:CheckPath -and (Test-Path -LiteralPath $script:CheckPath)){Remove-Item -LiteralPath $script:CheckPath;$script:CheckPath=$null}}})
$detect.Add_Click({try{$path=Resolve-ConfigPath $llama.Text;if(-not(Test-Path -LiteralPath $path -PathType Leaf)){throw 'Selecione um executável existente.'};$status.Text='Consultando versão, parâmetros e GPUs do llama selecionado...';Begin-Task ('-probe "'+$path+'" -backend '+@('server','cli')[$backend.SelectedIndex]) 'probe'}catch{$status.Text='Falha: '+$_}})
function Check-And-Save($action){try{if(-not $script:Profile){throw 'Clique em Detectar GPUs antes de testar ou salvar.'};if($output.Value -gt $ctx.Value/2){throw 'Saída máxima deve ser menor ou igual à metade do contexto.'};if($layers.Text -notmatch '^(auto|all|\d+)$'){throw 'Camadas GPU: informe auto, all ou um inteiro não negativo.'};foreach($box in @($main,$compiler)){if(-not(Test-Path -LiteralPath (Resolve-ConfigPath $box.Text) -PathType Leaf)){throw 'Selecione os dois arquivos GGUF existentes.'}};$selected=Resolve-ConfigPath $llama.Text;$directory=Split-Path -Parent $selected;$gm=Selected-Device $gpuMain;$gc=Selected-Device $gpuCompiler;$serverPath=Join-Path $directory 'llama-server.exe';$cliPath=Join-Path $directory 'llama-cli.exe';if($backend.SelectedIndex -eq 0){$serverPath=$selected}else{$cliPath=$selected};$script:Pending=[ordered]@{backend=@('server','cli')[$backend.SelectedIndex];llama_server=(Portable-Path $serverPath);llama_cli=(Portable-Path $cliPath);main_model=(Portable-Path $main.Text);compiler_model=(Portable-Path $compiler.Text);main_device=$gm.id;compiler_device=$gc.id;main_device_name=$gm.name;compiler_device_name=$gc.name;data_dir=(Portable-Path $data.Text);listen=('127.0.0.1:'+ $port.Value);mode=$mode.SelectedItem;context_tokens=[int]$ctx.Value;output_tokens=[int]$output.Value;gpu_layers=$layers.Text};$script:CheckPath=Join-Path $base ('.hame-config-check-'+[Guid]::NewGuid().ToString('N')+'.json');[IO.File]::WriteAllText($script:CheckPath,($script:Pending|ConvertTo-Json -Depth 8),(New-Object Text.UTF8Encoding($false)));$script:SaveAction=$action;$status.Text='Validando os arquivos e parâmetros necessários, sem carregar os modelos...';Begin-Task ('-config "'+$script:CheckPath+'" -check') 'check'}catch{$status.Text='Falha: '+$_}}
$test.Add_Click({Check-And-Save 'test'});$save.Add_Click({Check-And-Save 'save'});$start.Add_Click({Check-And-Save 'start'});$cancel.Add_Click({$form.Close()})
$form.Add_FormClosing({if($script:Task -and -not $script:Task.HasExited){$script:Task.Kill()};$timer.Stop();if($script:CheckPath -and (Test-Path -LiteralPath $script:CheckPath)){Remove-Item -LiteralPath $script:CheckPath}})
if($SelfTest){
 Device-Choices @(@{id='Vulkan9';name='Mock GPU';memory='4096 MiB'});if($gpuMain.Items.Count -ne 3){throw 'Device selector self-test failed'};Select-Device $gpuMain 'none';if((Selected-Device $gpuMain).id -ne 'none'){throw 'CPU selection failed'}
 if($EnginePath -and $llama.Text){
  Begin-Task ('-probe "'+(Resolve-ConfigPath $llama.Text)+'" -backend '+@('server','cli')[$backend.SelectedIndex]) 'probe'
  $deadline=[DateTime]::UtcNow.AddSeconds(50)
  while($script:Task -and [DateTime]::UtcNow -lt $deadline){[Windows.Forms.Application]::DoEvents();Start-Sleep -Milliseconds 50}
  if(-not $script:Profile){throw ('Async detection failed: '+$status.Text)}
  Check-And-Save 'test'
  $deadline=[DateTime]::UtcNow.AddSeconds(50)
  while($script:Task -and [DateTime]::UtcNow -lt $deadline){[Windows.Forms.Application]::DoEvents();Start-Sleep -Milliseconds 50}
  if(-not $status.Text.StartsWith('Configuração válida')){throw ('Async configuration check failed: '+$status.Text)}
 }
 $form.Dispose();Write-Output 'CONFIGURATOR SELFTEST PASS';exit 0
}
$null=$form.ShowDialog();$form.Dispose()
if($Managed -and -not $script:SavedAndStart){exit 2}
