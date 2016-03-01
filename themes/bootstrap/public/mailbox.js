var baseURL = window.location.protocol + '//' + window.location.host;
var navBarOffset = 75;
var mediumDeviceWidth = 980;
var messageListMargin = 275;
var clipboard = null;

// onDocChange is called each time we load partials into the DOM
function onDocChange() {
  // Bootstrap tooltips
  $('[data-toggle="tooltip"]').tooltip()

  // Clipboard functionality
  if (clipboard != null) {
    clipboard.destroy();
  }
  clipboard = new Clipboard('.btn-clipboard');
  clipboard.on('success', function(el) {
    flashTooltip(el.trigger, 'Copied!');
    el.clearSelection();
  });
  clipboard.on('error', function(el) {
    flashTooltip(el.trigger, 'Copy Failed!');
  });
}

// flashTooltip temporarily changes the text of a tooltip
function flashTooltip(el, text) {
  var prevText = $(el).attr('data-original-title');
  $(el).attr('data-original-title', text).tooltip('show');
  $(el).attr('data-original-title', prevText);
}

function messageLoaded(responseText, textStatus, XMLHttpRequest) {
  if (textStatus == "error") {
    alert("Failed to load message, server said:\n" + responseText);
    return;
  }
  onDocChange();
  var top = $('#message-container').offset().top - navBarOffset;
  $(window).scrollTop(top);
}

function listLoaded() {
  onDocChange();
  $('.listEntry').click(
      function() {
        $('.listEntry').removeClass("disabled");
        $(this).addClass("disabled");
        $('#message-content').load('/mailbox/' + mailbox + '/' + this.id, messageLoaded);
        selected = this.id;
      }
      )
    $("#message-list").slideDown();
  if (selected != "") {
    $("#" + selected).click();
    selected = "";
  }
}

function loadList() {
  $('#message-list').load('/mailbox/' + mailbox, listLoaded);
}

function reloadList() {
  $('#message-list').hide();
  loadList();
}

function windowResize() {
  if ($(window).width() > mediumDeviceWidth) {
    var content_height = $(window).height() - messageListMargin;
    $('#message-list-wrapper').height(content_height).addClass("message-list-scroll");
  } else {
    $('#message-list-wrapper').height('auto').removeClass("message-list-scroll");
  }
}

function listInit() {
  $("#message-list").hide();
  windowResize();
  $(window).resize(windowResize);
  loadList();
}

function deleteMessage(id) {
  $('#message-content').empty();
  $.ajax({
    type: 'DELETE',
    url: '/mailbox/' + mailbox + '/' + id,
    success: reloadList
  })
}

function htmlView(id) {
  window.open('/mailbox/' + mailbox + '/' + id + "/html", '_blank',
      'width=800,height=600,' +
      'menubar=yes,resizable=yes,scrollbars=yes,status=yes,toolbar=yes');
}

function messageSource(id) {
  window.open('/mailbox/' + mailbox + '/' + id + "/source", '_blank',
      'width=800,height=600,' +
      'menubar=no,resizable=yes,scrollbars=yes,status=no,toolbar=no');
}

function toggleMessageLink(id) {
  var url = baseURL + '/link/' + mailbox + '/' + id;
  $('#link-input-control').val(url);
  $('#link-row').slideToggle();
}

