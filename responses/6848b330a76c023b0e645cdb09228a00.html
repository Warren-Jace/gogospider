<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta http-equiv="Content-Type" content="text/html; charset=iso-8859-1" />
<title>ajax test</title>
<link href="styles.css" rel="stylesheet" type="text/css" />
<script type="text/javascript">
	var httpreq = null;	

	function SetContent(XML) {
		var items = XML.getElementsByTagName('items').item(0).getElementsByTagName('item');
		var inner = '<ul>';
		for(i=0; i<items.length; i++){
			inner = inner + '<li><a href="javascript:getInfo(\'' + items[i].attributes.item(0).value + '\', \'' + items[i].attributes.item(1).value + '\')">' + items[i].firstChild.nodeValue + '</a></li>';
		}
		
		inner = inner + '</ul>'
		
		cd = document.getElementById('contentDiv');
		cd.innerHTML = inner;
		
		id = document.getElementById('infoDiv');
		id.innerHTML = '';
	}

	function httpCompleted() {
		if (httpreq.readyState==4 && httpreq.status==200) {						
			SetContent(httpreq.responseXML);
			httpreq = null;
		}		
	}
	
	function SetInfo(XML) {
		var ii = XML.getElementsByTagName('iteminfo').item(0);
		var inner = '';
		
		inner = inner + '<p><strong>' + ii.getElementsByTagName('name').item(0).firstChild.nodeValue + '</strong></p>';
		
		pict  = ii.getElementsByTagName('picture');
		if(pict.length>0){
			inner = inner + '<img src="../showimage.php?file=' + pict.item(0).firstChild.nodeValue + '"/>';
		}
		
		descs = ii.getElementsByTagName('description');
		for (i=0; i<descs.length; i++){
			inner = inner + '<p>' + descs.item(i).firstChild.nodeValue + '</p>';
		}
		
		id = document.getElementById('infoDiv');
		id.innerHTML = inner;
	}
	
	function httpInfoCompleted() {		
		if (httpreq.readyState==4 && httpreq.status==200) {
			SetInfo(httpreq.responseXML);
			httpreq = null;
		}		
	}
	
	function loadSomething(what) {			
		getHttpRequest();		
		httpreq.open('GET', what, true);		
		httpreq.send('');
	}
	
	function getInfo(where, which) {
		getHttpRequest();		
		httpreq.onreadystatechange = httpInfoCompleted;
		if (where=='infotitle'){
			httpreq.open('POST', where+'.php', true);
			httpreq.setRequestHeader('content-type', 'application/x-www-form-urlencoded');
			httpreq.send('id='+which);
		}
		else {
			httpreq.open('GET', where+'.php?id='+which, true);
			httpreq.send('');
		}
	}
	
	function xmlCompleted () {
		if (httpreq.readyState==4 && httpreq.status==200) {
			xd = document.getElementById('xmlDiv');
			xd.innerHTML = httpreq.responseText;
			httpreq = null;
		}	
	}
	
	function sendXML () {
		getHttpRequest();
		httpreq.onreadystatechange = xmlCompleted;
		httpreq.open('POST', 'showxml.php');
		httpreq.setRequestHeader('content-type', 'text/xml');
		httpreq.send('<xml><node name="nodename1">nodetext1</node><node name="nodename2">nodetext2</node></xml>');
	}
	
	function getHttpRequest() {
		// free the curent one
		if (httpreq!=null){
			httpreq.abort();
			httpreq = null;
		}		

    if( window.XMLHttpRequest ) {
			httpreq = new XMLHttpRequest();
			if (httpreq.overrideMimeType) {
					httpreq.overrideMimeType('text/xml');
			}
    } else if(ActiveXObject) {
     	httpreq = new ActiveXObject("Msxml2.XMLHTTP");
		}
			httpreq.onreadystatechange = httpCompleted;
	}
	
	function SetMyCookie() {
		document.cookie = "mycookie=3";
		alert('A cookie was set by JavaScript.');
	}
</script>
</head>
<body>
<table border="0" cellpadding="3" width="500" align="center">
  <tr>
    <td class="bordered">
			<a href="javascript:loadSomething('artists.php');">artists</a> |
			<a href="javascript:loadSomething('categories.php');">categories</a> | 
			<a href="#" onclick="loadSomething('titles.php')">titles</a> |
			<a href="#" onclick="sendXML()">send xml</a> |
			<a href="#" onclick="SetMyCookie()">setcookie</a>
		</td>
  </tr>
  <tr>
    <td>
			<div id="contentDiv">
				&nbsp;
			</div>
		</td>
  </tr>
	<tr>
    <td>
			<div id="infoDiv">
				&nbsp;
			</div>
		</td>
  </tr>
	
	<tr>
    <td>
			<div id="xmlDiv">
				&nbsp;
			</div>
		</td>
  </tr>
</table>
</body>
</html>
