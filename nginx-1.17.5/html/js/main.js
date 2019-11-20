var Message = class {
	constructor(el) {
		this.act = "message";
		if (el !== undefined) {
			this.login = el.login;
			this.userid = el.userid;
			this.body = el.body;
			this.time = el.time;
			this.to = el.to;
		} else {
			this.login = "Анон";
			this.userid = 0;
			this.body = "";
			this.time = 0;
			this.to = 0;
		}
	}
}

var MessageList = class {
	constructor(ws) {
		var that = this;
		this.messages = new Array();

		this.send = function(model) {
			ws.send(JSON.stringify(model));
		};
	}
}

function getClientByName(name)
{
	result = false;
	for (var i = 0; i < clients.length; i++) {
		if(clients[i].name == name)
		{
			result = clients[i];
		}
	}
	return result;
}

function getClientById(id)
{
	result = "";
	for (var i = 0; i < clients.length; i++) {
		if(clients[i].id == id)
		{
			result = clients[i];
		}
	}
	return result;
}

function addMessage(msg)
{
	if(msg.userid == _USERID) {
		$($("#msgs .chat-message")[0]).addClass('self')
		$($("#msgs .chat-message")[0]).removeClass('server')
	} else if(msg.userid == 0) {
		$($("#msgs .chat-message")[0]).addClass('server')
		$($("#msgs .chat-message")[0]).removeClass('self')
	} else {
		$($("#msgs .chat-message")[0]).removeClass('server')
		$($("#msgs .chat-message")[0]).removeClass('self')
	}
	$($("#msgs .chat-message")[0]).attr("data-userid",msg.userid)
	$($("#msgs .chat-message")[0]).find(".c-icon").html(msg.login.substr(-msg.login.length,1))
	client = getClientById(msg.userid);
	console.log(client);
	if(client != false)	{
		color = client.color;
	} else {
		color = "#d1d1da";
	}
	$($("#msgs .chat-message")[0]).find(".c-icon").css("background-color", color)
	for (var emoji in _EMOJI) {
		msg.body = msg.body.replace(_EMOJI[emoji], "<i class='ic_emoji ic_"+emoji+"'></i>")
	}
	$($("#msgs .chat-message")[0]).find(".c-text").html(msg.body)
	var date = new Date();
	date.setTime(msg.time * 1000);
	$($("#msgs .chat-message")[0]).find(".c-date").html(("0" + date.getHours()).slice(-2)+":"+("0" + date.getMinutes()).slice(-2))
	$("#msgs").append($($("#msgs").find(".chat-message")[0]).prop('outerHTML'));
	setTimeout(function(){
		$("#msgs").scrollTop($("#msgs")[0].scrollHeight)
	}, 50)
}

function showRoomTitle(room){
	if(room == 0)
	{
		$("#chat-title").find("h2").html("Общая комната");
		desc = "участников";
		if(clients.length.toString().slice(-1) == 1)
		{
			desc = "участник";
		}
		else if(clients.length.toString().slice(-1) > 1 && clients.length.toString().slice(-1) < 5)
		{
			desc = "участника";
		}
		else if(clients.length.toString().slice(-1) > 10 && clients.length.toString().slice(-1) < 20)
		{
			desc = "участников";
		}
		$("#chat-title").find("p.desc").html(clients.length+" "+desc);
	}
	else
	{
		$("#chat-title").find("h2").html(getClientById(room).name);
		$("#chat-title").find("p.desc").html("Приватный чат");
	}
}

function showMessages(room)
{
	$("#msgs .chat-message").each(function(index, el) {
		if(index > 0)
		{
			$(el).remove();
		}
	});
	for(var i = 0; i < list.messages.length; i++)
	{
		msg = list.messages[i];
		dist = msg.to;
		if(dist == _USERID){
			dist = msg.userid
		}
		if(_ACTIVECHAT == dist)
		{
			addMessage(msg);
		}
	}
	$("#users .user-block").removeClass('active');
	$("#users .user-block[data-id='"+room+"']").addClass('active');
	showRoomTitle(room);
}

function showLastMessages()
{
	for(var i = 0; i < list.messages.length; i++)
	{
		msg = list.messages[i];
		dist = msg.to;
		if(dist == _USERID){
			dist = msg.userid
		}
		$("#users .user-block[data-id='"+dist+"']").find(".u-desc").html(msg.login+": "+msg.body.slice(0,25)+(msg.body.length>20?"..":""));
	}
}

function startWebSocket(location)
{
	ws = new WebSocket(location);
	list = new MessageList(ws);
	ws.onclose = function(){
		console.log("WS Close");
		$("#pushModal").removeClass('success');
		$("#pushModal").removeClass('show');
		$("#pushModal").addClass('show');
		$("#pushModal").addClass('error');
		$("#pushModal").find('.header').html('Ошибка');
		$("#pushModal").find('.text').html('Невозможно подключится к серверу');
		setTimeout(function(){
		  $("#pushModal").removeClass('success');
		  $("#pushModal").removeClass('error');
		  $("#pushModal").removeClass('show');
		  startWebSocket(location);
		}, 5000)
	}
	ws.onopen = function(){
		if(_USERLOGIN != "")
		{
			list.send({act:"login", login:_USERLOGIN})
			$("#pushModal").removeClass('error');
			$("#pushModal").removeClass('show');
			$("#pushModal").addClass('show');
			$("#pushModal").addClass('success');
			$("#pushModal").find('.header').html('Подключено');
			$("#pushModal").find('.text').html('Подключение к серверу прошло успешно');
			setTimeout(function(){
			  $("#pushModal").removeClass('success');
			  $("#pushModal").removeClass('error');
			  $("#pushModal").removeClass('show');
			}, 3000)
		}
	}
	ws.onmessage = function(e) {
		var model = JSON.parse(e.data);
		console.log(model.act);
		if(model.act == "loginResult")
		{
			console.log(model);
			if(model.body == "True")
			{
				$("#login_form").parent().removeClass("active");
				$("#overlay").removeClass("active");
				_USERLOGIN = model.login;
				_USERID = model.userid;
				setTimeout(function(){ showMessages(0); }, 100);
			}
			else
			{
				alert(model.body);
			}
		}
		else if(model.act == "message")
		{
			var msg = new Message(model);
			list.messages.push(msg);
			dist = msg.to;
			if(dist == _USERID){
				dist = msg.userid
			}
			if(_ACTIVECHAT == dist)
			{
				addMessage(msg);
			}
			showLastMessages();
		}
		else if(model.act == "msgResult")
		{
			if(model.body != "True")
			{
				alert(model.body);
			}
		}
		else if(model.act == "clientsListUpdate")
		{
			clients = JSON.parse(model.body);

			$("#users .user-block").each(function(index, el) {
				if(index > 0 && $(el).attr("data-id") != 0)
				{
					$(el).remove();
				}
			});
			for (var i = 0; i < clients.length; i++) {
				$($("#users .user-block")[0]).attr("data-id", clients[i].id);
				$($("#users .user-block")[0]).find(".u-icon").html(clients[i].name.substr(-clients[i].name.length,1))
				$($("#users .user-block")[0]).find(".u-icon").css("background-color", clients[i].color)
				$($("#users .user-block")[0]).find(".u-name").html(clients[i].name)
				if(clients[i].name == _USERLOGIN){
					console.log(clients[i].name+" == "+_USERLOGIN)
					$($("#users .user-block")[0]).find(".u-desc").html('<i>Это вы.</i>')
				}
				else
				{
					$($("#users .user-block")[0]).find(".u-desc").html('')
				}
				$("#users").append($($("#users").find(".user-block")[0]).prop('outerHTML'));
			}
			showLastMessages();
			if(_ACTIVECHAT == 0)
			{
				showRoomTitle(0);
			}
		}
		else if(model.act == "serverMsg")
		{
			var msg = new Message(model);
			msg.userid = 0;
			msg.to = 0;
			msg.login = "Server";
			list.messages.push(msg);
			if(_ACTIVECHAT == 0)
			{
				addMessage(msg);
			}
			showLastMessages();
		}
	};
}

_EMOJI = {
	"devil": "}:)",
	"smile": ":)",
	"cool": "B)",
	"frown": ":(",
	"tongue": ":P",
	"wink": ";)"
}

$(document).ready(function() {
	_USERLOGIN = "";
	_USERID = "";
	_ACTIVECHAT = 0;
	clients = new Array();
	startWebSocket("ws://localhost:8080/entry"); //192.168.3.178

	$("#chatInput").on("submit", function(){
		var data = $("#chatInput").serializeArray();
		list.send({act:"message", login:_USERLOGIN, userid:_USERID, body:data[0].value, to:parseInt(_ACTIVECHAT)})
		$("#chatInput")[0].reset()
		return false;
	})

	$("#login_form").on("submit", function(){
		var data = $("#login_form").serializeArray();
		list.send({act:"login", login:data[0].value, userid:(_USERID!=""?_USERID:0)})
		$("#chatInput")[0].reset()
		return false;
	})

	$("#users").on("click", ".user-block", function(){
		chatid = $(this).attr("data-id");
		if(_ACTIVECHAT != chatid && chatid != _USERID)
		{
			_ACTIVECHAT = chatid;
			showMessages(_ACTIVECHAT);
		}
	})

	$("#emojibut").on("click", function(){
		$("#emoji-block").toggleClass('active');
	})

	for (var emoji in _EMOJI) {
		$("#emoji-block").append("<i class='ic_emoji ic_"+emoji+"' data-emoji='"+emoji+"'></i>");
	}

	$("#emoji-block").on("click", ".ic_emoji", function(){
		$("#inputField").val($("#inputField").val()+" "+_EMOJI[$(this).attr("data-emoji")]+" ");
		$("#emoji-block").removeClass('active');
	})
});