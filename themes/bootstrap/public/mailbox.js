function messageLoaded(responseText, textStatus, XMLHttpRequest) {
  if (textStatus == "error") {
    alert("Failed to load message, server said:\n" + responseText);
  } else {
    window.scrollTo(0,0);
  }
}

function listLoaded() {
  $('.listEntry').click(
      function() {
        $('.listEntry').removeClass("disabled");
        $(this).addClass("disabled");
        $('#emailContent').load('/mailbox/' + mailbox + '/' + this.id, messageLoaded);
        selected = this.id;
      }
      )
    $("#messageList").slideDown();
  if (selected != "") {
    $("#" + selected).click();
    selected = "";
  }
}

function loadList() {
  $('#messageList').load('/mailbox/' + mailbox, listLoaded);
}

function reloadList() {
  $('#messageList').hide();
  loadList();
}

function listInit() {
  $("#messageList").hide();
  loadList();
}

function deleteMessage(id) {
  $('#emailContent').empty();
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

