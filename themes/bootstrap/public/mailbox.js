var baseURL = window.location.protocol + '//' + window.location.host;
var navBarOffset = 75;
var mediumDeviceWidth = 980;
var messageListMargin = 275;
var clipboard = null;
var messageListScroll = false;
var messageListData = null;

// clearMessageSearch resets the message list search
function clearMessageSearch() {
  $('#message-search').val('');
  updateMessageSearch();
}

// deleteMessage sends a delete request for a message
function deleteMessage(id) {
  $('#message-content').empty();
  $.ajax({
    type: 'DELETE',
    url: '/api/v1/mailbox/' + mailbox + '/' + id,
    success: loadList
  })
}

// Delete the mailbox
function deleteMailBox() {
  cont = confirm("Are you sure you want delete the Mailbox?")
  if (cont == false) {
    return;
  }
  $.ajax({
    type: 'DELETE',
    url: '/api/v1/mailbox/' + mailbox,
    success: loadList
  })
}

// flashTooltip temporarily changes the text of a tooltip
function flashTooltip(el, text) {
  var prevText = $(el).attr('data-original-title');
  $(el).attr('data-original-title', text).tooltip('show');
  $(el).attr('data-original-title', prevText);
}

// htmlView pops open another window for viewing message as HTML
function htmlView(id) {
  window.open('/mailbox/' + mailbox + '/' + id + "/html", '_blank',
      'width=800,height=600,' +
      'menubar=yes,resizable=yes,scrollbars=yes,status=yes,toolbar=yes');
}

// loadList loads the message list for this mailbox via AJAX
function loadList() {
  $('#message-list').hide().empty();
  $.ajax({
    dataType: "json",
    url: '/api/v1/mailbox/' + mailbox,
    success: function(data) {
      messageListData = data.reverse();
      // Render list
      $('#message-list').loadTemplate($('#list-entry-template'), data);
      $('.message-list-entry').click(onMessageListClick);
      // Reveal and select current message
      $("#message-list").slideDown();
      if (selected != "") {
        $("#" + selected).click();
        selected = "";
      }
      onDocumentChange();
      updateMessageSearch();
    }
  });
}

// makeDelay creates a call-back timer that prevents itself from being
// stacked
function makeDelay(ms) {
  var timer = 0;
  return (function(callback) {
    clearTimeout (timer);
    timer = setTimeout(callback, ms);
  });
}

// messageSource pops open another window for message source
function messageSource(id) {
  window.open('/mailbox/' + mailbox + '/' + id + "/source", '_blank',
      'width=800,height=600,' +
      'menubar=no,resizable=yes,scrollbars=yes,status=no,toolbar=no');
}

// toggleMessageLink shows/hids the message link URL form
function toggleMessageLink(id) {
  var url = baseURL + '/link/' + mailbox + '/' + id;
  $('#link-input-control').val(url);
  $('#link-row').slideToggle();
}

// onDocumentChange is called each time we load partials into the DOM
function onDocumentChange() {
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

// onDocumentReady is called by mailbox/index.html to initialize
function onDocumentReady() {
  // Prevent search and resize handlers being called too often
  var searchDelay = makeDelay(200);
  var resizeDelay = makeDelay(100);
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
  $("#message-list").hide();
  onWindowResize();
  $(window).resize(function() {
    resizeDelay(onWindowResize);
  });
  $('#message-search').on('change keyup', function(el) {
    searchDelay(updateMessageSearch);
  });
  loadList();
}

// onMessageListClick is triggered by clicks on the message list
function onMessageListClick() {
  $('.message-list-entry').removeClass("disabled");
  $(this).addClass("disabled");
  $('#message-content').load('/mailbox/' + mailbox + '/' + this.id, onMessageLoaded);
  selected = this.id;
}

// onMessageLoaded is called each time a new message is shown
function onMessageLoaded(responseText, textStatus, XMLHttpRequest) {
  if (textStatus == "error") {
    alert("Failed to load message, server said:\n" + responseText);
    return;
  }
  onDocumentChange();
  var top = $('#message-container').offset().top - navBarOffset;
  $(window).scrollTop(top);
}

// onWindowResize handles special cases when window is resized
function onWindowResize() {
  if ($(window).width() > mediumDeviceWidth) {
    var content_height = $(window).height() - messageListMargin;
    var messageListWrapper = $('#message-list-wrapper');
    messageListWrapper.height(content_height);
    if (!messageListScroll) {
      messageListScroll = true;
      messageListWrapper.addClass("message-list-scroll");
    }
  } else {
    if (messageListScroll) {
      messageListScroll = false;
      $('#message-list-wrapper').height('auto').removeClass("message-list-scroll");
    }
  }
}

// updateMessageSearch compares the message list subjects and senders against
// the search string and hides entries that don't match
function updateMessageSearch() {
  var criteria = $('#message-search').val();
  if (criteria.length < 2) {
    $('.message-list-entry').show();
    return;
  }
  criteria = criteria.toLowerCase();
  for (i=0; i<messageListData.length; i++) {
    entry = messageListData[i];
    if ((entry.subject.toLowerCase().indexOf(criteria) > -1) ||
        (entry.from.toLowerCase().indexOf(criteria) > -1)) {
      // Match
      $('#' + entry.id).show();
    } else {
      $('#' + entry.id).hide();
    }
  }
}

