param([string]$ConfigPath = (Join-Path $PSScriptRoot 'config.json'), [switch]$Managed, [switch]$SelfTest, [string]$EnginePath = "", [string]$RenderPath = "")
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
[Windows.Forms.Application]::EnableVisualStyles()
$ConfigPath = [IO.Path]::GetFullPath($ConfigPath)
$base = Split-Path -Parent $ConfigPath
$exe = Join-Path $PSScriptRoot 'HAME.exe'
if($EnginePath){$exe=$EnginePath}
$form = New-Object Windows.Forms.Form
$form.Text = 'HAME 1.02 - Configurador'
$form.AutoScaleMode = [Windows.Forms.AutoScaleMode]::None
$form.ClientSize = New-Object Drawing.Size(940,800)
$form.StartPosition = 'CenterScreen'
$form.FormBorderStyle = 'Sizable'
$form.AutoScroll = $true
$form.AutoScrollMinSize = New-Object Drawing.Size(940,800)
$area=[Windows.Forms.Screen]::PrimaryScreen.WorkingArea
$form.ClientSize=New-Object Drawing.Size([Math]::Min(940,$area.Width-60),[Math]::Min(800,$area.Height-100))
$form.MinimumSize=New-Object Drawing.Size(650,500)
$form.MaximizeBox = $true
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

# Keep existing controls and event handlers; organize them into two tabs.
$tabs=New-Object Windows.Forms.TabControl;$tabs.SetBounds(10,10,905,565)
$general=New-Object Windows.Forms.TabPage;$general.Text='Motor e modelos'
$memory=New-Object Windows.Forms.TabPage;$memory.Text='Gerenciamento de Memória e Execução'
$tabs.TabPages.Add($general);$tabs.TabPages.Add($memory)
$outside=@($status,$test,$save,$start,$cancel)
foreach($control in @($form.Controls)){if($outside -notcontains $control){$general.Controls.Add($control)}}
$form.Controls.Add($tabs)
$status.SetBounds(20,590,835,120);$test.SetBounds(20,745,130,32);$save.SetBounds(450,745,120,32);$start.SetBounds(585,745,155,32);$cancel.SetBounds(755,745,100,32)
function Move-Memory($control,$x,$y,$width=120){$memory.Controls.Add($control);$control.SetBounds($x,$y,$width,28)}
function Find-Label($text){return @($general.Controls|Where-Object {$_.GetType().Name -eq 'Label' -and $_.Text -eq $text})[0]}
function Mem-Label($text,$x,$y,$width=195){$label=New-Object Windows.Forms.Label;$label.Text=$text;$label.SetBounds($x,$y,$width,28);$memory.Controls.Add($label);return $label}
function Mem-Number($x,$y,$min,$max,$value){$n=New-Object Windows.Forms.NumericUpDown;$n.Minimum=$min;$n.Maximum=$max;$n.Value=$value;$n.SetBounds($x,$y,120,28);$memory.Controls.Add($n);return $n}
Move-Memory (Find-Label 'Modo de execução') 10 28 195
Move-Memory $mode 210 25 610
$mode.Items.Clear();$null=$mode.Items.Add('Economia de memória (Constrained)');$null=$mode.Items.Add('Alto desempenho (Resident)');$mode.SelectedIndex=0
$modeDescription=Mem-Label 'Constrained: mantém o principal ativo e alterna modelos somente ao compactar.' 10 65 810;$modeDescription.Height=48
Move-Memory (Find-Label 'Contexto (tokens)') 10 135 190
Move-Memory $ctx 210 132
$null=Mem-Label 'Gatilho de compactação (%)' 430 135 215
$threshold=Mem-Number 650 132 10 99 90
$null=Mem-Label 'Reserva para resposta' 10 180
$reserve=Mem-Number 210 177 128 1048576 1024
Move-Memory (Find-Label 'Saída máxima') 430 180 195
Move-Memory $output 650 177
$null=Mem-Label 'Máximo da memória (tokens)' 10 225 200
$memoryMax=Mem-Number 210 222 128 65536 1024
$null=Mem-Label 'Reserva de template' 430 225 195
$templateReserve=Mem-Number 650 222 16 65536 128
$null=Mem-Label 'Política de memória' 10 270
$policy=New-Object Windows.Forms.ComboBox;$policy.DropDownStyle='DropDownList';$policy.SetBounds(210,267,205,30);$memory.Controls.Add($policy);foreach($value in @('Automática','Sob demanda','Híbrida')){$null=$policy.Items.Add($value)};$policy.SelectedIndex=2
Move-Memory (Find-Label 'Camadas GPU') 430 270 195
Move-Memory $layers 650 267
$prefill=New-Object Windows.Forms.CheckBox;$prefill.Text='Pré-carregar o contexto compilado (somente llama-server)';$prefill.Checked=$true;$prefill.SetBounds(10,315,805,28);$memory.Controls.Add($prefill)
$compatibility=Mem-Label 'Resident exige slots, descarte de cache e validação real. Use Testar modo selecionado.' 10 353 810;$compatibility.Height=48
$null=Mem-Label 'Sessão para status/compactar' 10 410 200
$session=New-Object Windows.Forms.TextBox;$session.SetBounds(210,407,610,28);$memory.Controls.Add($session)
$sessionHint=Mem-Label 'Em branco: conversa ativa no HAME já aberto. O histórico visual não é apagado.' 10 440 810
$runtime=New-Object Windows.Forms.Button;$runtime.Text='Consultar contexto';$runtime.SetBounds(10,480,160,32);$memory.Controls.Add($runtime)
$manual=New-Object Windows.Forms.Button;$manual.Text='Compactar agora';$manual.SetBounds(185,480,155,32);$memory.Controls.Add($manual)
$full=New-Object Windows.Forms.CheckBox;$full.Text='Recuperação completa';$full.SetBounds(350,482,230,28);$memory.Controls.Add($full)
$modeTest=New-Object Windows.Forms.Button;$modeTest.Text='Testar modo selecionado';$modeTest.SetBounds(590,480,230,32);$memory.Controls.Add($modeTest)
$portLabel=Find-Label 'Porta do chat';$portLabel.SetBounds(20,397,190,25);$port.SetBounds(215,394,120,28)
$mode.Add_SelectedIndexChanged({if($mode.SelectedIndex -eq 1){$modeDescription.Text='Resident: mantém os dois modelos e invalida a slot do principal sem descarregar os pesos.'}else{$modeDescription.Text='Constrained: mantém o principal ativo e alterna os modelos somente quando a compactação é necessária.'}})
$publicKeys=@('automatic','on_demand','hybrid')


$extraLabel=New-Object Windows.Forms.Label;$extraLabel.Text='Argumentos extras (principal)';$extraLabel.SetBounds(20,433,185,45);$general.Controls.Add($extraLabel)
$extra=New-Object Windows.Forms.TextBox;$extra.Multiline=$true;$extra.ScrollBars='Vertical';$extra.MaxLength=8192;$extra.SetBounds(215,430,610,52);$extra.AccessibleName='Argumentos extras do modelo principal';$general.Controls.Add($extra)
$notice.SetBounds(20,497,810,38)
$general.AutoScroll=$true
$general.AutoScrollMinSize=New-Object Drawing.Size(835,620)
$memory.AutoScroll=$true
$memory.AutoScrollMinSize=New-Object Drawing.Size(835,530)
$mmLabel=New-Object Windows.Forms.Label;$mmLabel.Text='Projetor mmproj (opcional)';$mmLabel.SetBounds(20,542,190,28);$general.Controls.Add($mmLabel)
$mmproj=New-Object Windows.Forms.TextBox;$mmproj.SetBounds(215,540,480,28);$general.Controls.Add($mmproj)
$browseMM=New-Object Windows.Forms.Button;$browseMM.Text='Procurar...';$browseMM.SetBounds(710,539,115,32);$general.Controls.Add($browseMM)
$mmHint=New-Object Windows.Forms.Label;$mmHint.Text='Server: fotos são salvas com descrição YAML. CLI: anexos não disponíveis.';$mmHint.SetBounds(20,578,805,32);$general.Controls.Add($mmHint)
$script:DeviceRows=@(); $script:Profile=$null; $script:Task=$null; $script:AfterTask=''; $script:SaveAction=''; $script:Pending=$null; $script:CheckPath=$null; $script:SavedAndStart=$false
function Device-Choices($devices) { $script:DeviceRows=@(@{id='auto';name='';text='Automático (prioridade do llama; CPU se indisponível)'},@{id='none';name='';text='Somente CPU (sem offload para GPU)'})
 foreach($d in $devices){$script:DeviceRows+=@{id=$d.id;name=$d.name;text=($d.id+' — '+$d.name+' — '+$d.memory)}}
 if(@($devices).Count -gt 0){$script:DeviceRows[0].text='Automático — '+$devices[0].id+' — '+$devices[0].name}else{$script:DeviceRows[0].text='Automático — CPU se o llama não anunciar GPU'}
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

$reserve.Value=[Math]::Max(128,[int]$output.Value)
if($ctx.Value -lt 4096){$memoryMax.Value=[int]($ctx.Value/4)}
if($script:Original.compaction_threshold_percent){$threshold.Value=$script:Original.compaction_threshold_percent}
if($script:Original.reserve_tokens){$reserve.Value=$script:Original.reserve_tokens}
if($script:Original.template_reserve_tokens){$templateReserve.Value=$script:Original.template_reserve_tokens}
if($script:Original.compiled_memory_max_tokens){$memoryMax.Value=$script:Original.compiled_memory_max_tokens}
if($script:Original.ContainsKey('prefill_compiled_context')){$prefill.Checked=[bool]$script:Original.prefill_compiled_context}
if($script:Original.memory_policy){$index=[Array]::IndexOf($publicKeys,[string]$script:Original.memory_policy);if($index -ge 0){$policy.SelectedIndex=$index}}
if($backend.SelectedIndex -eq 1){$prefill.Checked=$false;$prefill.Enabled=$false;$modeDescription.Text='CLI por tarefas: carrega e encerra cada chamada; não mantém pesos residentes.'}

if($script:Original.main_mmproj){$mmproj.Text=$script:Original.main_mmproj}
if($script:Original.main_extra_args){$extra.Text=$script:Original.main_extra_args}
$chooseFile={param($box,$filter);$dialog=New-Object Windows.Forms.OpenFileDialog;$dialog.Filter=$filter;if($dialog.ShowDialog($form)-eq 'OK'){$box.Text=$dialog.FileName};$dialog.Dispose()}
$browseLlama.Add_Click({&$chooseFile $llama 'Executável llama (*.exe)|*.exe'})
$browseMM.Add_Click({&$chooseFile $mmproj 'Projetor GGUF (*.gguf)|*.gguf'})
$browseMain.Add_Click({&$chooseFile $main 'Modelos GGUF (*.gguf)|*.gguf'})
$browseCompiler.Add_Click({&$chooseFile $compiler 'Modelos GGUF (*.gguf)|*.gguf'})
$browseData.Add_Click({$dialog=New-Object Windows.Forms.FolderBrowserDialog;if($dialog.ShowDialog($form)-eq 'OK'){$data.Text=$dialog.SelectedPath};$dialog.Dispose()})
$backend.Add_SelectedIndexChanged({$script:Profile=$null;Device-Choices @();if($llama.Text){$sibling=Join-Path (Split-Path -Parent $llama.Text) ('llama-'+@('server','cli')[$backend.SelectedIndex]+'.exe');if(Test-Path -LiteralPath $sibling){$llama.Text=$sibling}};if($backend.SelectedIndex -eq 1){$mode.SelectedIndex=0;$mode.Enabled=$false;$prefill.Checked=$false;$prefill.Enabled=$false;$modeDescription.Text='CLI por tarefas: carrega e encerra cada chamada; não mantém pesos residentes.';$compatibility.Text='CLI por tarefas: Resident e prefill residente não disponíveis. Tokenizer auxiliar é detectado ao lado do CLI.'}else{$mode.Enabled=$true;$prefill.Enabled=$true};$status.Text='Motor alterado. Detectando GPUs e versão...';if($autoDetect -and -not $SelfTest){$autoDetect.Start()}})
$autoDetect=New-Object Windows.Forms.Timer;$autoDetect.Interval=800
$autoDetect.Add_Tick({$autoDetect.Stop();if($script:Task){$autoDetect.Start();return};if($llama.Text -and (Test-Path -LiteralPath (Resolve-ConfigPath $llama.Text) -PathType Leaf)){$detect.PerformClick()}})
$llama.Add_TextChanged({$script:Profile=$null;if(-not $SelfTest){$autoDetect.Stop();$autoDetect.Start()}})
function Set-Busy($busy){foreach($c in @($backend,$llama,$detect,$browseLlama,$main,$compiler,$browseMain,$browseCompiler,$gpuMain,$gpuCompiler,$data,$browseData,$mode,$port,$ctx,$output,$layers,$extra,$mmproj,$browseMM,$threshold,$reserve,$templateReserve,$memoryMax,$policy,$prefill,$runtime,$manual,$modeTest,$test,$save,$start)){$c.Enabled=-not $busy};if($backend.SelectedIndex -eq 1){$mode.Enabled=$false;$prefill.Enabled=$false}else{if($script:Profile -and -not $script:Profile.resident_candidate){$mode.SelectedIndex=0;$mode.Enabled=$false}}}
function Begin-Task($arguments,$kind){if($script:Task){return};$info=New-Object Diagnostics.ProcessStartInfo;$info.FileName=$exe;$info.Arguments=$arguments;$info.WorkingDirectory=$base;$info.UseShellExecute=$false;$info.CreateNoWindow=$true;$info.RedirectStandardOutput=$true;$info.RedirectStandardError=$true;$info.StandardOutputEncoding=[Text.Encoding]::UTF8;$info.StandardErrorEncoding=[Text.Encoding]::UTF8;$process=New-Object Diagnostics.Process;$process.StartInfo=$info;$null=$process.Start();$script:ReadOut=$process.StandardOutput.ReadToEndAsync();$script:ReadErr=$process.StandardError.ReadToEndAsync();$script:Task=$process;$script:AfterTask=$kind;Set-Busy $true;$timer.Start()}
$timer=New-Object Windows.Forms.Timer;$timer.Interval=200
$timer.Add_Tick({if(-not $script:Task -or -not $script:Task.HasExited){return};$timer.Stop();$process=$script:Task;$text=$script:ReadOut.Result;$errors=$script:ReadErr.Result;$code=$process.ExitCode;$script:Task=$null;$process.Dispose();Set-Busy $false;try{
 if($code -ne 0){throw ($text+"`r`n"+$errors)}
 if($script:AfterTask -eq 'probe'){$p=$text|ConvertFrom-Json;if($p.error){throw $p.error};$script:Profile=$p;Device-Choices $p.devices;if($p.resident_candidate){$mode.Enabled=$true;$compatibility.Text='Resident: capacidades anunciadas. Testar modo selecionado verifica slot/prefill e processos reais.'}else{$mode.SelectedIndex=0;$mode.Enabled=$false;$compatibility.Text='Resident indisponível para este backend. Constrained permanece disponível.'};Select-Device $gpuMain $script:Original.main_device;Select-Device $gpuCompiler $script:Original.compiler_device;$status.Text=$p.version+"`r`n"+([string]$p.devices.Count)+' GPU(s) detectada(s). CPU permanece disponível. Alterações de versão são verificadas por capacidades.'}
 elseif($script:AfterTask -eq 'runtime'){$r=$text|ConvertFrom-Json;$u=$r.status.context_usage;$status.Text='Modo: '+$r.mode+' | Política: '+$r.memory_policy+"`r`nSessão: "+$r.session_id+"`r`nEtapa: "+$r.status.stage;if($u.context_tokens){$status.Text+="`r`nContexto: "+$u.prompt_tokens+'/'+$u.context_tokens+' tokens; reserva '+$u.reserved_response_tokens+' + template '+$u.template_reserve_tokens+'; orçamento '+$u.budget_percent+'%';if(-not $u.exact_prompt){$status.Text+=' (template protegido por reserva)'}}}
 elseif($script:AfterTask -eq 'compact'){$r=$text|ConvertFrom-Json;$status.Text='Memória renovada. Histórico preservado. Compactações: '+$r.compactions+"`r`nOrçamento atual: "+$r.usage.budget_percent+'%'}
 elseif($script:AfterTask -eq 'mode'){$status.Text='Teste real do modo concluído. '+$text}
 elseif($script:AfterTask -eq 'check'){$status.Text='Configuração válida. '+$text;if($script:SaveAction -ne 'test'){$json=$script:Pending|ConvertTo-Json -Depth 8;[IO.File]::WriteAllText($ConfigPath,$json,(New-Object Text.UTF8Encoding($false)));$status.Text+=' Salva em '+$ConfigPath;if($script:SaveAction -eq 'start'){$script:SavedAndStart=$true;$form.Close();if(-not $Managed){Start-Process -FilePath $exe -ArgumentList ('-config "'+$ConfigPath+'" -open') -WorkingDirectory $base}}}}
 }catch{$status.Text='Falha: '+$_}finally{if($script:CheckPath -and (Test-Path -LiteralPath $script:CheckPath)){Remove-Item -LiteralPath $script:CheckPath;$script:CheckPath=$null}}})
$detect.Add_Click({try{$path=Resolve-ConfigPath $llama.Text;if(-not(Test-Path -LiteralPath $path -PathType Leaf)){throw 'Selecione um executável existente.'};$status.Text='Consultando versão, parâmetros e GPUs do llama selecionado...';Begin-Task ('-probe "'+$path+'" -backend '+@('server','cli')[$backend.SelectedIndex]) 'probe'}catch{$status.Text='Falha: '+$_}})
function Check-And-Save($action){try{if(-not $script:Profile){throw 'Clique em Detectar GPUs antes de testar ou salvar.'};if($reserve.Value -lt $output.Value){throw 'A reserva deve cobrir a saída máxima.'};if($memoryMax.Value+$reserve.Value+$templateReserve.Value -ge $ctx.Value){throw 'Memória compilada e reservas devem deixar espaço na janela.'};if($output.Value -gt $ctx.Value/2){throw 'Saída máxima deve ser menor ou igual à metade do contexto.'};if($layers.Text -notmatch '^(auto|all|\d+)$'){throw 'Camadas GPU: informe auto, all ou um inteiro não negativo.'};foreach($box in @($main,$compiler)){if(-not(Test-Path -LiteralPath (Resolve-ConfigPath $box.Text) -PathType Leaf)){throw 'Selecione os dois arquivos GGUF existentes.'}};$selected=Resolve-ConfigPath $llama.Text;$directory=Split-Path -Parent $selected;$gm=Selected-Device $gpuMain;$gc=Selected-Device $gpuCompiler;$serverPath=Join-Path $directory 'llama-server.exe';$cliPath=Join-Path $directory 'llama-cli.exe';if($backend.SelectedIndex -eq 0){$serverPath=$selected}else{$cliPath=$selected};$script:Pending=[ordered]@{backend=@('server','cli')[$backend.SelectedIndex];main_extra_args=$extra.Text;main_mmproj=$(if($mmproj.Text.Trim()){Portable-Path $mmproj.Text}else{''});llama_server=(Portable-Path $serverPath);llama_cli=(Portable-Path $cliPath);main_model=(Portable-Path $main.Text);compiler_model=(Portable-Path $compiler.Text);main_device=$gm.id;compiler_device=$gc.id;main_device_name=$gm.name;compiler_device_name=$gc.name;data_dir=(Portable-Path $data.Text);listen=('127.0.0.1:'+ $port.Value);mode=@('constrained','resident')[$mode.SelectedIndex];compaction_threshold_percent=[int]$threshold.Value;reserve_tokens=[int]$reserve.Value;template_reserve_tokens=[int]$templateReserve.Value;compiled_memory_max_tokens=[int]$memoryMax.Value;memory_policy=$publicKeys[$policy.SelectedIndex];prefill_compiled_context=[bool]$prefill.Checked;context_tokens=[int]$ctx.Value;output_tokens=[int]$output.Value;gpu_layers=$layers.Text};foreach($key in $script:Original.Keys){if(-not $script:Pending.Contains($key)){$script:Pending[$key]=$script:Original[$key]}};$script:CheckPath=Join-Path $base ('.hame-config-check-'+[Guid]::NewGuid().ToString('N')+'.json');[IO.File]::WriteAllText($script:CheckPath,($script:Pending|ConvertTo-Json -Depth 8),(New-Object Text.UTF8Encoding($false)));$script:SaveAction=$action;$status.Text='Validando os arquivos e parâmetros necessários, sem carregar os modelos...';if($action -eq 'mode'){$status.Text='Carregando modelos e verificando o modo. Pode demorar; não inicia outra conversa.';Begin-Task ('-config "'+$script:CheckPath+'" -test-mode') 'mode'}else{Begin-Task ('-config "'+$script:CheckPath+'" -check') 'check'}}catch{$status.Text='Falha: '+$_}}
$tips=New-Object Windows.Forms.ToolTip;$tips.SetToolTip($extra,'Exemplo: --threads 8. Use aspas duplas em valores com espaços. Não executa shell. Parâmetros controlados pelo HAME não podem ser sobrescritos. Para ferramentas, configure tools e tool_cwd no config.json; veja docs/API.md.');$tips.SetToolTip($policy,'Automática: gatilho; Sob demanda: comando/ação, bloqueia overflow; Híbrida: gatilho ou /recuperar. A compactação manual permanece disponível.');$modeTest.Add_Click({Check-And-Save 'mode'});$runtime.Add_Click({Begin-Task ('-config "'+$ConfigPath+'" -memory-status') 'runtime'});$manual.Add_Click({$recoverFlag='';if($full.Checked){$recoverFlag=' -recover'};Begin-Task ('-config "'+$ConfigPath+'" -compact -session "'+$session.Text+'"'+$recoverFlag) 'compact'});$test.Add_Click({Check-And-Save 'test'});$save.Add_Click({Check-And-Save 'save'});$start.Add_Click({Check-And-Save 'start'});$cancel.Add_Click({$form.Close()})
$form.Add_FormClosing({if($script:Task -and -not $script:Task.HasExited){$script:Task.Kill()};$timer.Stop();if($script:CheckPath -and (Test-Path -LiteralPath $script:CheckPath)){Remove-Item -LiteralPath $script:CheckPath}})
$form.Add_Shown({$autoDetect.Start()})
if($SelfTest){
 Device-Choices @(@{id='Vulkan9';name='Mock GPU';memory='4096 MiB'});if($gpuMain.Items.Count -ne 3){throw 'Device selector self-test failed'};Select-Device $gpuMain 'none';if((Selected-Device $gpuMain).id -ne 'none'){throw 'CPU selection failed'}
 if($EnginePath -and $llama.Text){
  $form.ShowInTaskbar=$false;$form.StartPosition='Manual';$form.Location=New-Object Drawing.Point(-10000,-10000);$form.Show()
  $deadline=[DateTime]::UtcNow.AddSeconds(50)
  while(-not $script:Profile -and [DateTime]::UtcNow -lt $deadline){[Windows.Forms.Application]::DoEvents();Start-Sleep -Milliseconds 50}
  if(-not $script:Profile){throw ('Async detection failed: '+$status.Text)}
  Check-And-Save 'test'
  $deadline=[DateTime]::UtcNow.AddSeconds(50)
  while($script:Task -and [DateTime]::UtcNow -lt $deadline){[Windows.Forms.Application]::DoEvents();Start-Sleep -Milliseconds 50}
  if(-not $status.Text.StartsWith('Configuração válida')){throw ('Async configuration check failed: '+$status.Text)}
 }
 if($threshold.Value -ne 90 -and -not $script:Original.compaction_threshold_percent){throw 'Migration threshold failed'};if($reserve.Value -lt $output.Value){throw 'Migration reserve failed'};foreach($key in @('compaction_threshold_percent','reserve_tokens','template_reserve_tokens','compiled_memory_max_tokens','memory_policy','prefill_compiled_context')){if($EnginePath -and $llama.Text -and -not $script:Pending.Contains($key)){throw ('Missing saved memory field '+$key)}};if($script:Original.ContainsKey('custom_preference') -and $script:Pending.custom_preference -ne $script:Original.custom_preference){throw 'Unknown preference lost'};if($script:Original.ContainsKey('main_extra_args') -and $script:Pending -and $script:Pending.main_extra_args -ne $script:Original.main_extra_args){throw 'Startup arguments lost'};if($RenderPath){$form.ShowInTaskbar=$false;$form.StartPosition='Manual';$form.Location=New-Object Drawing.Point(-10000,-10000);$form.Show();[Windows.Forms.Application]::DoEvents();$form.PerformLayout();$bitmap=New-Object Drawing.Bitmap($form.ClientSize.Width,$form.ClientSize.Height);$form.DrawToBitmap($bitmap,(New-Object Drawing.Rectangle(0,0,$form.ClientSize.Width,$form.ClientSize.Height)));$bitmap.Save($RenderPath,[Drawing.Imaging.ImageFormat]::Png);$bitmap.Dispose();$tabs.SelectedTab=$memory;[Windows.Forms.Application]::DoEvents();$bitmap=New-Object Drawing.Bitmap($form.ClientSize.Width,$form.ClientSize.Height);$form.DrawToBitmap($bitmap,(New-Object Drawing.Rectangle(0,0,$form.ClientSize.Width,$form.ClientSize.Height)));$memoryPath=Join-Path ([IO.Path]::GetDirectoryName($RenderPath)) (([IO.Path]::GetFileNameWithoutExtension($RenderPath))+'-memory.png');$bitmap.Save($memoryPath,[Drawing.Imaging.ImageFormat]::Png);$bitmap.Dispose()};$form.Dispose();Write-Output 'CONFIGURATOR SELFTEST PASS';exit 0
}
$null=$form.ShowDialog();$autoDetect.Stop();$autoDetect.Dispose();$form.Dispose()
if($Managed -and -not $script:SavedAndStart){exit 2}
