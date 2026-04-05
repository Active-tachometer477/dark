package dark

import "strings"

// mcpBaseCSS provides reset styles and MCP theme variable defaults.
const mcpBaseCSS = `*,*::before,*::after{box-sizing:border-box}body{margin:0;font-family:var(--mcp-ui-font-family,system-ui,-apple-system,sans-serif);background:var(--mcp-ui-bg-primary,#fff);color:var(--mcp-ui-text-primary,#000)}`

// mcpAppBridgeJS is the MCP App Bridge: handles postMessage JSON-RPC 2.0
// communication with the host application following the MCP Apps protocol.
const mcpAppBridgeJS = `(function(){'use strict';
var _rid=0,_pending={},_onToolResult=null,_onThemeChange=null,_ctx=null;

function send(m){window.parent.postMessage(m,'*');}

function request(method,params){
  return new Promise(function(ok,ng){
    var id=++_rid;
    _pending[id]={ok:ok,ng:ng};
    send({jsonrpc:'2.0',id:id,method:method,params:params||{}});
  });
}

window.addEventListener('message',function(ev){
  var m=ev.data;
  if(!m||m.jsonrpc!=='2.0')return;

  if(m.id&&_pending[m.id]){
    var p=_pending[m.id];delete _pending[m.id];
    m.error?p.ng(new Error(m.error.message||'RPC error')):p.ok(m.result);
    return;
  }

  if(m.method==='ui/notifications/tool-result'){
    if(_onToolResult)_onToolResult(m.params);
  }else if(m.method==='ui/notifications/host-context-changed'){
    if(m.params){
      if(_ctx)Object.assign(_ctx,m.params);
      if(m.params.styles&&m.params.styles.variables){
        var v=m.params.styles.variables;
        for(var k in v)document.documentElement.style.setProperty(k,v[k]);
      }
      if(m.params.theme){
        document.documentElement.setAttribute('data-theme',m.params.theme);
        if(_onThemeChange)_onThemeChange(m.params.theme);
      }
    }
  }
});

window.__dark_bridge={
  ready:function(){
    request('ui/initialize',{
      protocolVersion:'2026-01-26',
      appCapabilities:{},
      appInfo:{name:'dark-mcp-app',version:'1.0.0'}
    }).then(function(r){
      _ctx=r.hostContext||{};
      if(_ctx.styles&&_ctx.styles.variables){
        var v=_ctx.styles.variables;
        for(var k in v)document.documentElement.style.setProperty(k,v[k]);
      }
      if(_ctx.styles&&_ctx.styles.css&&_ctx.styles.css.fonts){
        var s=document.createElement('style');
        s.textContent=_ctx.styles.css.fonts;
        document.head.appendChild(s);
      }
      if(_ctx.theme){
        document.documentElement.setAttribute('data-theme',_ctx.theme);
        if(_onThemeChange)_onThemeChange(_ctx.theme);
      }
      send({jsonrpc:'2.0',method:'ui/notifications/initialized'});
      _autoResize();
    }).catch(function(){});
  },
  callServerTool:function(name,args){
    return request('tools/call',{name:name,arguments:args||{}});
  },
  readResource:function(uri){
    return request('resources/read',{uri:uri});
  },
  sendMessage:function(text){
    return request('ui/message',{role:'user',content:{type:'text',text:text}});
  },
  openLink:function(url){
    return request('ui/open-link',{url:url});
  },
  updateContext:function(content,structured){
    return request('ui/update-model-context',{content:content,structuredContent:structured});
  },
  onToolResult:function(fn){_onToolResult=fn;},
  onThemeChange:function(fn){_onThemeChange=fn;},
  getHostContext:function(){return _ctx;}
};

function _autoResize(){
  var _last=0;
  function notify(){
    var h=document.documentElement.scrollHeight||document.body.scrollHeight;
    if(h&&h!==_last){_last=h;send({jsonrpc:'2.0',method:'ui/notifications/size-changed',params:{height:h}});}
  }
  new MutationObserver(notify).observe(document.body,{childList:true,subtree:true,attributes:true});
  new ResizeObserver(notify).observe(document.body);
  notify();
}
})();`

// assembleMCPAppHTML builds a self-contained HTML document for an MCP App.
// The HTML is a static app shell — data arrives via postMessage from the host.
func assembleMCPAppHTML(css string, clientJS string) string {
	var b strings.Builder
	b.Grow(len(css) + len(clientJS) + len(mcpAppBridgeJS) + 512)

	b.WriteString("<!DOCTYPE html><html><head><meta charset=\"UTF-8\">")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">")
	b.WriteString("<meta name=\"color-scheme\" content=\"light dark\">")
	b.WriteString("<style>")
	b.WriteString(mcpBaseCSS)
	b.WriteString("</style>")
	if css != "" {
		b.WriteString("<style>")
		b.WriteString(css)
		b.WriteString("</style>")
	}
	b.WriteString("</head><body>")

	b.WriteString("<div id=\"app\"></div>")

	b.WriteString("<script type=\"module\">")
	b.WriteString(mcpAppBridgeJS)
	b.WriteString("</script>")

	b.WriteString("<script type=\"module\">")
	b.WriteString(clientJS)
	b.WriteString("</script>")

	b.WriteString("</body></html>")

	return b.String()
}
