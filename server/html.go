package main

func getIndexPage() string {

	html :=
		`
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8" />
			<meta name="description" content="Dashboard" />
			<meta name="viewport" content="width=device-width, initial-scale=1.0" />
			<meta http-equiv="X-UA-Compatible" content="IE=edge" />
			<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
			<link rel="stylesheet" type="text/css" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" >
			<link rel="stylesheet" type="text/css" href="https://cdnjs.cloudflare.com/ajax/libs/toastr.js/latest/toastr.min.css" >
			<style>
				.modal-xl {
					width: 90%;
					max-width: 1200px;
				}
			</style>
	
	
			<script src="https://ajax.googleapis.com/ajax/libs/jquery/2.1.3/jquery.min.js"></script>
			<script src="https://cdnjs.cloudflare.com/ajax/libs/Chart.js/2.4.0/Chart.min.js"></script>
		</head>
		<body>
			<div class="container">
				<div id=wrapper style="padding: 0 0 40px 0;">
					<div class="row">
						<div class="col-md-12">
							<h1 class="text-center">Charts</h1>
							<canvas id="myChart" width="1417" height="708" class="chartjs-render-monitor" style="display: block; width: 1417px; height: 708px;"></canvas>
						</div>
					</div>
					<div class="row">
						<div class="col-md-6">
							<h3 class="text-center">Choose Chart</h3>
							<input type="radio" class="chart-radio" id=hour-chart name="chart-radio"> 1 Hour <br/>
							<input type="radio" class="chart-radio" id=day-chart name="chart-radio"> 24 Hour <br/>
							<input type="radio" class="chart-radio" id=week-chart name="chart-radio"> Week
						</div>
						<div class="col-md-6">
							<h3 class="text-center">Warnings</h3>
							<div id="warning-section" style="color:red; font-size:16px"></div>
							<div id="device-message" style="color:red; font-size:16px"></div>
						</div>
					</div>
					<div class="row">
						<div class="col-md-6">
							<h3 class="text-center">Record Mode</h3>
							<div class="row">
								<form id="record-form">
									<div class="col-md-12">
									<input type="radio" class="record-mode" name="record-mode" value="true"> Record | 
									<input type="radio" class="record-mode" name="record-mode" value="false"> Don't Record
										<hr /> 
									</div>
									<div class="col-md-12">
										<input type="checkbox" class="record-device" id=record-all name="record-device-all" value="All"> All 
										<hr /> 
									</div>
									{{ range $deviceName, $device := .deviceCenter.Devices }}
										<div class="col-md-12">
											<input type="checkbox" class="record-device" name="record-device" value="{{ $deviceName }}"> {{ $deviceName }}
		
											{{ if $device.IsRecording }}
												<span class="pull-right"> Mode: <span class="mode-text" data-device-name="{{ $deviceName }}" style="color:green">Recording</span></span>
											{{ else }}
												<span class="pull-right"> Mode: <span class="mode-text" data-device-name="{{ $deviceName }}" style="color:red">Not Recording</span></span>
											{{ end }}
										</div> 
									{{ end }}
									<div class="col-md-12">
										<div class="form-group">
											<input type="text" name="password" class="form-control" id="record-password" placeholder="Password">
										</div>
										<button type="button" id="record-submit" class="btn btn-primary">Submit</button>
									</div>
								</form>
							</div>
						</div>
						<div class="col-md-6">
							<h3 class="text-center">New Set</h3>
							<div class="row">
								<form id="new-set-form">
									<div class="col-md-12">
										<input type="checkbox" id=new-set-all name="new-set-all"> All 
										<hr />
									</div>
									{{ range $deviceName, $device := .deviceCenter.Devices }}
										<div class="col-md-12">
											<input type="checkbox" class="new-set" name="new-set" value="{{ $deviceName }}"> {{ $deviceName }}
										</div> 
									{{ end }}
									<div class="col-md-12">
										<div class="form-group">
											<input type="text" name="password" class="form-control" id="new-set-password" placeholder="Password">
										</div>
										<button type="button" id="new-set-submit" name="new-set-submit" class="btn btn-primary">Submit</button>
									</div>
								</form>
							</div>
						</div>
					</div>
					<div class="row" style="margin: 25px 0 0 0;">
						<div class="col-md-12">
							<h2 class="text-center">Device Table</h2>
							<table id=device-table class="table">
								<tr>
									<!-- <th><input type="checkbox" id="check-all-sets" /></th> -->
									<th>Device Name</th>
									<th># of Sets</th>
									<th>Lastest Set Time</th>
									<th></th>
								</tr>
								{{ range $deviceName, $device := .deviceCenter.Devices }}
									<tr class="table-row" data-device-name="{{ $deviceName }}">
										<td>{{ $deviceName }}</td>
										<td class="num-of-sets">{{ $device.SetNum }}</td>
										<td class="latest-set">
											{{ if $device.LatestSetTime }}
												{{ $device.LatestSetTime }}
											{{ else }}
												N/A
											{{ end }}
										</td>
										<td>
											<form class="form-inline device-form">
												<input type="hidden" class="device-name" name="deviceName" value="{{ $deviceName }}" />
												<div class="form-group">
													<input type="text" name="password" placeholder="Password" class="form-control password">
												</div>
												<button type="button" class="btn btn-primary device-submit">Submit</button>
												<a class="btn btn-success hidden-download">Download</a>
											</form>
											</div>
										</td>
									</tr>
								{{end}}
							</table>
							<form id="all-devices-form" class="form-inline all-device-form">
								<div class="form-group">
									<input type="text" name="password" placeholder="Password" class="form-control password">
									<button type="button" id=all-devices-submit class="btn btn-primary">Submit</button>
									<a class="btn btn-success hidden-download">Download</a>
								</div>
								<!-- <button type="button" class="btn btn-primary device-submit">Submit</button> -->
								<!-- <button type="button" class="btn btn-primary all-device-submit">Submit</button> -->
								
							</form>
						</div>
					</div>
				</div>
			</div>
	
			<div class="modal fade" id="modal" tabindex="-1" role="dialog">
				<div class="modal-dialog modal-xl" role="document">
					<div class="modal-content">
						<div class="modal-header">
							<button type="button" class="close" data-dismiss="modal" aria-label="Close"><span aria-hidden="true">&times;</span></button>
							<h4 class="modal-title">Device Sets</h4>
						</div>
						<form id="modal-form" action="">
							<input type="hidden" id="hidden-device-name" name="deviceName"/>
							<div class="modal-body" id=modal-body>
										
							</div>
							<div class="modal-footer" id=modal-footer>
								<input type="text" class="form-control" name="password" placeholder="Password" id=modal-password />
								<button class="btn btn-block btn-primary" type="button" style="margin: 20px 0 0 0;" id="modal-download">Download</button>
							</div>
						</form>
					</div><!-- /.modal-content -->
				</div><!-- /.modal-dialog -->
			</div><!-- /.modal -->
	
			<div style="display:none;" id="modal-section">
				<div class="row" id="device-section">
					
				</div>
			</div>
	
		</body>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/moment.js/2.18.1/moment.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/toastr.js/latest/toastr.min.js"></script>
		<script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js"></script>
	
		<script>
			function updateChartHandler(timeMeasure){
				$.ajax({
					url: "/update-chart-handler/",
					method: "GET",
					data: {timeMeasure: timeMeasure},
					success: function(result){
				
					},
					error: function(xhr, status, stringMessage){
						toastr.error(xhr.responseText);
					}
				});
			}
	
			function hideDownloadHandler(){
				$(".hidden-download").each(function(i, item){
					$(this).hide();
				});
	
				$(".hidden-download").on("click", function(e){
					$(this).hide();
				});
			}
	
			function generateAllDevicesTarHandler(){
				$("#all-devices-submit").on("click", function(e){
					var serialize = $("#all-devices-form").serialize();
					console.log(serialize);
					$.ajax({
						url: "/generate-all-devices-tar/",
						method: "POST",
						data: serialize,
						success: function(result){
							jsonResult = JSON.parse(result);
							$("#all-devices-form").find(".password").val("");
							var $download = $("#all-devices-form").find(".hidden-download");
							$download.attr("href", "/download-tar/?fileName=" + jsonResult.file);
							$download.attr("download", "AllDevices.tar.gz");
							$download.show();
						},
						error: function(xhr, status, message){
							if (xhr.status == 403){
								toastr.error(xhr.responseText)
							}
							else{
								alert("Server error");
							}
						}
					});
				});
			}
	
			function generateDeviceTarHandler(){
				$(".device-submit").on("click", function(e){
					var deviceName = $(this).closest(".device-form").find(".device-name").val();
					var serialize = $(this).closest(".device-form").serialize();
					var self = $(this);
					console.log("device name: " + deviceName);
					$.ajax({
						url: "/generate-device-tar/",
						method: "POST",
						data: serialize,
						success: function(result){
							jsonResult = JSON.parse(result);
							self.closest(".device-form").find(".password").val("");
							var $download = self.closest(".device-form").find(".hidden-download");
							$download.attr("href", "/download-tar/?fileName=" + jsonResult.file);
							$download.attr("download", deviceName + ".tar.gz");
							$download.show();
						},
						error: function(xhr, status, message){
							if (xhr.status == 403){
								toastr.error(xhr.responseText)
							}
							else{
								alert("Server error");
							}
						}
					});
				});
			}
	
			function modalDownloadButtonHandler(){
				// $("#modal-download").on("click", function(e){
				//     var serialize = $("#modal-form").serialize();
				//     console.log(serialize);
				//     if(serialize.includes("setName")){
				//         $.ajax({
				//             url: "/download-device-sets/",
				//             method: "POST",
				//             data: serialize,
				//             success: function(result){
				//                 $("#modal").modal('hide');
				//             },
				//             error: function(xhr, status, message){
				//                 console.log(xhr);
				//                 console.log(status);
				//                 console.log(message);
				//                 if(xhr.status == 403){
				//                     toastr.error(xhr.responseText);
				//                 }
				//                 else if(xhr.status == 406){
				//                     toastr.error(xhr.responseText);
				//                 }
				//                 else{
				//                     alert("Server error");
				//                 }
				//             }
				//         });
				//     }
				//     else{
				//         toastr.warning("Must select at least one set");
				//     }
				// });
			}
	
			function downloadButtonHandler(){
				$("#download").on("click", function(e){
					var deviceName = $(this).val();
					var setNames = [];
					$(".device-set").each(function(i, item){
						if($(this).is(":checked")){
							setNames.push($(this).val());
						}
					});
	
					$.ajax({
						url: "/download-devices/",
						method: "POST",
						dataType: "json",
						data:{
							deviceName: deviceName,
							setNames: setNames
						},
						success: function(result){
							$("#modal").modal('hide');
						},
						error: function(xhr, message){
							alert("Server error");
						}
					})
				});
			}
	
			function deviceModalHandler(){
				$(".device-set-button").on("click", function(e){
					var fileNameList = JSON.parse($(this).attr("data-file-name-list")),
						deviceName = $(this).val(),
						html = '';
					
					for(var i = 0; i < fileNameList.length; i++){
						html += 
						'<div class="col-md-2">' +
							'<input type="checkbox" name="setName" class="modal-checkbox" value="'+ fileNameList[i] +'" /> ' + fileNameList[i] +
						'</div>';
					}
	
					console.log(html);
	
					$("#modal-body").html("");
					$("#device-section").html(html);
					$("#hidden-device-name").val(deviceName);
					$("#modal-body").append($("#modal-section").html());
					$("#modal").modal('toggle');
				});
			}
	
			function checkAllDeviceSetsHandler(){
				$("#check-all-sets").on("click", function(){
					if($(this).is(":checked")){
						$(".device-set").each(function(i, item){
							$(this).prop("checked", true);
						});
					}
					else{
						$(".device-set").each(function(i, item){
							$(this).prop("checked", false);
						});
					}
				});
			}
	
			function newSetSubmitHandler(){
				$("#new-set-submit").on("click", function(e){
					var serialize = $("#new-set-form").serialize();
					$.ajax({
						url: "/new-set/",
						method: "POST",
						data: serialize,
						success: function(result){
							toastr.success("New set created for selected devices");
							$("#new-set-password").val("");
							$("#new-set-all").prop('checked', false);
							$(".new-set").each(function(i, item){
								$(this).prop('checked', false);
							});
							var result = JSON.parse(result);
							console.log(result);
	
							if(result.message != "" && result.message != null){
								$("#device-message").html(result.message);
							}
							else{
								$("#device-message").html("");
							}
	
							$(".table-row").each(function(i, item){
								var tableDeviceName = $(this).attr("data-device-name");
								for(var i = 0; i < result.chartArray.length; i++){
									if(result.devices[i].name == tableDeviceName){
										$(this).find(".num-of-sets").html(result.chartArray[i].numOfSets);
										$(this).find(".latest-set").html(result.chartArray[i].latestSet);
									}
								}
							});
							
							// $(".new-set-text").each(function(i, item){
							//     var deviceName = $(this).attr('data-new-set-device');
							//     var setNumber = result.deviceSet[deviceName];
							//     $(this).html(setNumber);
							// });
						},
						error: function(xhr, status, message){
							toastr.error(xhr.responseText);
						}
					})
				});
			}
	
			function statusesHandler(){
				$.ajax({
					url: "/update-status-handler/",
					success: function(result){
						var result = JSON.parse(result),
							displayString = "";
	
						if(Object.keys(result).length === 0){
							$("#warning-section").html("");
						}
						else{
							for (var property in result) {
								if (result.hasOwnProperty(property)) {
									displayString += "Last heard device '" + property + "' at " + moment(result[property]).format("MM/DD/YYYY H:mm:ss") + " <br />";
									console.log(result[property]);
								}
							}
							$("#warning-section").html(displayString);
						}
					},
					error: function(xhr, status, message){
						
					}
				});
			}
	
			function recordSubmitHandler(){
				$("#record-submit").on("click", function(e){
					var serialize = $("#record-form").serialize();
					$.ajax({
						url: "/record-mode-handler/",
						data: serialize,
						method: "POST",
						success: function(result){
							var result = JSON.parse(result);
							$(".mode-text").each(function(i, item){
								var deviceName = $(this).attr('data-device-name');
								console.log(deviceName);
								var recordMode = result[deviceName];
								console.log(recordMode);
								if(recordMode){
									$(this).css('color', 'green').html("Recording");
								}else{
									$(this).css('color', 'red').html("Not Recording");
								}
	
								$("#record-password").val("");
								$(".record-device").each(function(i, item){
									$(this).prop('checked', false);
								});
							});
							console.log(result);
							$("#warning-section").html("");
							$("#device-message").html("");
							toastr.success("Record mode has been changed for selected devices");
						},
						error: function(xhr, status, message){
							toastr.error(xhr.responseText);
						}
					});
				});
			}
	
			function substractSet(){
				$(".new-set-text").each(function(i, item){
					var setNumber = parseInt($(this).html()) - 1;
					$(this).html(setNumber);
				});
			}
	
			function newSetCheckboxHandler(){
				$("#new-set-all").on("click", function(e){
					if($(this).is(":checked")){
						$(".new-set").each(function(i, item){
							$(this).prop('checked', true);
						});
					}
					else{
						$(".new-set").each(function(i, item){
							$(this).prop('checked', false);
						});
					}
				});
			}
	
			function recordCheckboxHandler(){
				$("#record-all").on("click", function(e){
					if($(this).is(":checked")){
						$(".record-device").each(function(i, item){
							$(this).prop('checked', true);
						});
					}
					else{
						$(".record-device").each(function(i, item){
							$(this).prop('checked', false);
						});
					}
				});
			}
	
			function initChart(){
	
			}
	
			$(document).ready(function(){
				
				var MONTHS = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];
				var minutes = ["10", "20", "30", "40", "50", "60"];
				var hours = ["0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23"];
				var ctx = document.getElementById('myChart').getContext('2d');
				var config = {
					type: 'line',
					data: {
						labels: hours,
						datasets: [{
							label: "Device 1",
							backgroundColor: "#f44242",
							borderColor: "#f44242",
							data: [
								10,
								35,
								25,
								50,
								70,
								15,
								45
							],
							fill: false,
						}, {
							label: "Device 2",
							fill: false,
							backgroundColor: "#4141f4",
							borderColor: "#4141f4",
							data: [
								20,
								10,
								60,
								50,
								80,
								30,
								50
							],
						}]
					},
					options: {
						// responsive: true,
						title:{
							display:true,
							text:'24 Hour'
						},
						tooltips: {
							mode: 'index',
							intersect: false,
						},
						hover: {
							mode: 'nearest',
							intersect: true
						},
						scales: {
							xAxes: [{
								display: true,
								scaleLabel: {
									display: true,
									labelString: 'Time'
								}
							}],
							yAxes: [{
								display: true,
								scaleLabel: {
									display: true,
									labelString: 'Movement'
								}
							}]
						}
					}
				};
				var chart = new Chart(ctx, config);
			// window.myLine.update();
	
			// document.getElementById('randomizeData').addEventListener('click', function() {
			//     config.data.datasets.forEach(function(dataset) {
			//         dataset.data = dataset.data.map(function() {
			//             return randomScalingFactor();
			//         });
	
			//     });
	
			//     window.myLine.update();
			// });
	
			// var colorNames = Object.keys(window.chartColors);
			// document.getElementById('addDataset').addEventListener('click', function() {
			//     var colorName = colorNames[config.data.datasets.length % colorNames.length];
			//     var newColor = window.chartColors[colorName];
			//     var newDataset = {
			//         label: 'Dataset ' + config.data.datasets.length,
			//         backgroundColor: newColor,
			//         borderColor: newColor,
			//         data: [],
			//         fill: false
			//     };
	
			//     for (var index = 0; index < config.data.labels.length; ++index) {
			//         newDataset.data.push(randomScalingFactor());
			//     }
	
			//     config.data.datasets.push(newDataset);
			//     window.myLine.update();
			// });
	
			// document.getElementById('addData').addEventListener('click', function() {
			//     if (config.data.datasets.length > 0) {
			//         var month = MONTHS[config.data.labels.length % MONTHS.length];
			//         config.data.labels.push(month);
	
			//         config.data.datasets.forEach(function(dataset) {
			//             dataset.data.push(randomScalingFactor());
			//         });
	
			//         window.myLine.update();
			//     }
			// });
	
			// document.getElementById('removeDataset').addEventListener('click', function() {
			//     config.data.datasets.splice(0, 1);
			//     window.myLine.update();
			// });
	
			// document.getElementById('removeData').addEventListener('click', function() {
			//     config.data.labels.splice(-1, 1); // remove the label first
	
			//     config.data.datasets.forEach(function(dataset, datasetIndex) {
			//         dataset.data.pop();
			//     });
	
			//     window.myLine.update();
			// });
		
	
				$(".record-mode").first().prop('checked', true);
				$(".chart-radio").first().prop('checked', true);
				hideDownloadHandler();
				generateDeviceTarHandler();
				generateAllDevicesTarHandler();
				newSetCheckboxHandler();
				recordCheckboxHandler();
				recordSubmitHandler();
				newSetSubmitHandler();
				checkAllDeviceSetsHandler();
				modalDownloadButtonHandler();
				deviceModalHandler();
				substractSet();
				// getData("all");
				window.setInterval(statusesHandler, 3000);
			});
		</script>
	
	</html>
	`
	return html
}
