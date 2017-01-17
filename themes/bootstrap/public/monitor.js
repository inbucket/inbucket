var baseURL = window.location.protocol + '//' + window.location.host;

function startMonitor() {
  $.addTemplateFormatter({
    "date": function(value, template) {
      return moment(value).calendar();
    },
    "subject": function(value, template) {
      if (value == null || value.length == 0) {
        return "(No Subject)";
      }
      return value;
    }
  });

  var uri = '/api/v1/monitor/all/messages'
  var l = window.location;
  var url = ((l.protocol === "https:") ? "wss://" : "ws://") + l.host + uri
  var ws = new WebSocket(url);

  ws.addEventListener('message', function (e) {
    var msg = JSON.parse(e.data);
    msg['href'] = '/mailbox?name=' + msg.mailbox + '&id=' + msg.id;
    $('#monitor-message-list').loadTemplate(
        $('#message-template'),
        msg,
        { append: true });
  });
}

function messageClick(node) {
  var href = node.attributes['href'].value;
  var url = baseURL + href;
  window.location.assign(url);
}

function clearClick() {
  $('#monitor-message-list').empty();
}
