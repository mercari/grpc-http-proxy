$(document).ready(function() {
  var grpcServicesMap

  function updateGrpcServices(grpcServices) {
    $('#grpcServices h3').remove();

    $.each(grpcServices, function(index, service) {
      $('#grpcServices').append('<div><h3 class="grpcService">'+service+'</h3></div>')
    });
  }

  $.ajax({url: "/grpcServices", success: function(result){
    grpcServicesMap = result["grpc_service"]
    var grpcServices = Object.keys(grpcServicesMap)

    updateGrpcServices(grpcServices)
  }})

  $(document).on("click", "h3.grpcService" , function() {
    var serviceNameElm = $(this)
    var serviceName = serviceNameElm.text()

    serviceNameElm.parent().find('.serviceVersion').remove()

    var versions = grpcServicesMap[serviceName]
    var versionsName = Object.keys(versions)

    $.each(versionsName, function(index, version) {
      var url = versions[version][0]["Scheme"] + ":" + versions[version][0]["Opaque"]
      serviceNameElm.parent().append('<div class="serviceVersion"><p class="serviceVersionBar" style="display:inline;margin-left:10px"> - </p><p style="display:inline" class="serviceVersion">'+url+'</p><p class="serviceVersionTag" style="display:inline"> ('+version+')</p></div>')
    })
  })

  $(document).on("click", "p.serviceVersion" , function() {
    var serviceUrlElm = $(this)
    var serviceUrl = serviceUrlElm.text()

    serviceUrlElm.parent().find('.grpcSvc').remove()
    url = "/services?url=" + serviceUrl

    $.ajax({url: url, success: function(result){
      $.each(result, function(index, svc) {
        serviceUrlElm.parent().append('<div class="grpcSvc" ><p style="display:inline;margin-left:20px" class="grpcSvcBar"> - </p><p style="display:inline" class="grpcSvc">'+svc+'</p></div>')
      })
    }})
  })

  $(document).on("click", "p.grpcSvc" , function() {
    var grpcSvcElm = $(this)
    var grpcSvcName = $(this).text()
    var serviceUrlElm = grpcSvcElm.parent().parent().find('p.serviceVersion')
    var serviceUrl = serviceUrlElm.text()


    grpcSvcElm.parent().find('.grpcSvcMethod').remove()
    url = "/methods?url=" + serviceUrl + "&service=" + grpcSvcName

    $.ajax({url: url, success: function(result){
      $.each(result, function(index, method) {
        grpcSvcElm.parent().append('<div class="grpcSvcMethod"><p style="display:inline;margin-left:30px" class="grpcSvcMethodBar"> - </p><p style="display:inline" class="grpcSvcMethod">'+method+'</p></div>')
      })
    }})
  })

  $(document).on("click", "p.grpcSvcMethod" , function() {
    var grpcSvcMethodElm = $(this)
    var grpcSvcMethodName = grpcSvcMethodElm.text()

    var grpcSvcElm = grpcSvcMethodElm.parent().parent().find('p.grpcSvc')
    var grpcSvcName = grpcSvcElm.text()

    var serviceUrlElm = grpcSvcMethodElm.parent().parent().parent().find('p.serviceVersion')
    var serviceUrl = serviceUrlElm.text()

    console.log(serviceUrl)

    grpcSvcMethodElm.parent().find('.grpcSvcMethodField').remove()

    url = "/fields?url=" + serviceUrl + "&service=" + grpcSvcName + "&method=" + grpcSvcMethodName

    $.ajax({url: url, success: function(result){
      console.log(result)
      if (result.length == 0) {
        result = ["no field"]
      }
      $.each(result, function(index, field) {
        grpcSvcMethodElm.parent().append('<div class="grpcSvcMethodField"><p style="display:inline;margin-left:40px" class="grpcSvcMethodFieldBar"> - </p><p style="display:inline" class="grpcSvcMethodField">'+field+'</p></div>')
      })
    }})
  })

})
