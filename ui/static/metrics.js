// Inbucket Status Metrics
jQuery.ajaxSetup({ cache: false });
flashOn = jQuery.Color("rgba(255,255,0,1)");
flashOff = jQuery.Color("rgba(255,255,0,0)");
dataHist = new Object();

function timeFilter(seconds) {
  if (seconds < 60) {
    return seconds + " seconds";
  } else if (seconds < 3600) {
    return (seconds/60).toFixed(0) + " minute(s)";
  } else if (seconds < 86400) {
    return (seconds/3600).toFixed(1) + " hour(s)";
  }
  return (seconds/86400).toFixed(0) + " day(s)";
}

function sizeFilter(bytes) {
  if (bytes < 1024) {
    return bytes + " bytes";
  } else if (bytes < 1048576) {
    return (bytes/1024).toFixed(0) + " KB";
  } else if (bytes < 1073741824) {
    return (bytes/1048576).toFixed(2) + " MB";
  }
  return (bytes/1073741824).toFixed(2) + " GB";
}

function numberFilter(x) {
  var parts = x.toString().split(".");
  parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return parts.join(".");
}

function appendHistory(name, value) {
  var h = dataHist[name];
  if (! h) {
    h = new Array(0);
  }
  // Prevent array from growing
  if (h.length >= 60) {
    h = h.slice(1,60);
  }
  h.push(parseInt(value));
  dataHist[name] = h;
  el = $('#s-' + name);
  if (el) {
    el.sparkline(dataHist[name]);
  }
}

// Show spikes for numbers that only increase
function setHistoryOfActivity(name, value) {
  var h = value.split(",");
  var prev = parseInt(h[0]);
  for (i=0; i<h.length; i++) {
    var t = parseInt(h[i]);
    h[i] = t-prev;
    prev = t;
  }
  // First value will always be zero
  if (h.length > 0) {
    h = h.slice(1);
  }
  el = $('#s-' + name);
  if (el) {
    el.sparkline(h);
  }
}

// Show up/down for numbers that can decrease
function setHistoryOfCount(name, value) {
  var h = value.split(",");
  el = $('#s-' + name);
  if (el) {
    el.sparkline(h);
  }
}

function metric(name, value, filter, chartable) {
  if (chartable) {
    appendHistory(name, value);
  }
  if (filter) {
    value = filter(value);
  }
  var el = $('#m-' + name)
    if (el.text() != value) {
      el.text(value);
      el.css('background-color', flashOn);
      el.animate({ backgroundColor: flashOff }, 1500);
    }
}

function displayMetrics(data, textStatus, jqXHR) {
  // Non graphing
  metric('uptime', data.uptime, timeFilter, false);
  metric('retentionScanCompleted', data.retention.SecondsSinceScanCompleted, timeFilter, false);
  metric('retentionPeriod', data.retention.Period, timeFilter, false);

  // JavaScript history
  metric('memstatsSys', data.memstats.Sys, sizeFilter, true);
  metric('memstatsHeapAlloc', data.memstats.HeapAlloc, sizeFilter, true);
  metric('memstatsHeapSys', data.memstats.HeapSys, sizeFilter, true);
  metric('memstatsHeapObjects', data.memstats.HeapObjects, numberFilter, true);
  metric('smtpConnectsCurrent', data.smtp.ConnectsCurrent, numberFilter, true);
  metric('goroutinesCurrent', data.goroutines, numberFilter, true);
  metric('httpWebSocketConnectsCurrent', data.http.WebSocketConnectsCurrent, numberFilter, true);

  // Server-side history
  metric('smtpReceivedTotal', data.smtp.ReceivedTotal, numberFilter, false);
  setHistoryOfActivity('smtpReceivedTotal', data.smtp.ReceivedHist);
  metric('smtpConnectsTotal', data.smtp.ConnectsTotal, numberFilter, false);
  setHistoryOfActivity('smtpConnectsTotal', data.smtp.ConnectsHist);
  metric('smtpWarnsTotal', data.smtp.WarnsTotal, numberFilter, false);
  setHistoryOfActivity('smtpWarnsTotal', data.smtp.WarnsHist);
  metric('smtpErrorsTotal', data.smtp.ErrorsTotal, numberFilter, false);
  setHistoryOfActivity('smtpErrorsTotal', data.smtp.ErrorsHist);
  metric('retentionDeletesTotal', data.retention.DeletesTotal, numberFilter, false);
  setHistoryOfActivity('retentionDeletesTotal', data.retention.DeletesHist);
  metric('retainedCurrent', data.retention.RetainedCurrent, numberFilter, false);
  setHistoryOfCount('retainedCurrent', data.retention.RetainedHist);
  metric('retainedSize', data.retention.RetainedSize, sizeFilter, false);
  setHistoryOfCount('retainedSize', data.retention.SizeHist);
}

function loadMetrics() {
  // jQuery.getJSON( url [, data] [, success(data, textStatus, jqXHR)] )
  jQuery.getJSON('/debug/vars', null, displayMetrics);
}
