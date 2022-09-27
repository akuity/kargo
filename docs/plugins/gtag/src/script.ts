import type {PluginOptions} from './options';

export function script(opts: PluginOptions): string {
return `function enableGtag() {
  window['ga-disable-${opts.trackingID}'] = false;
  gtag('js', new Date());
  gtag(
    'config', 
    '${opts.trackingID}',
    {
      'anonymize_ip': ${opts.anonymizeIP ? 'true' : 'false'},
      'send_page_view': false
    }
  );
}

function disableGtag() {
  window['ga-disable-${opts.trackingID}'] = true;
}

function gtag(){
  dataLayer.push(arguments);
}

function deleteCookies(cookieconsent_name) {
  var keep = [cookieconsent_name];
  document.cookie.split(';').forEach(function(c) {
    c = c.split('=')[0].trim();
    if (!~keep.indexOf(c))
        document.cookie = c + '=;' + 'expires=Thu, 01 Jan 1970 00:00:00 UTC;path=/';
  });
};

window.addEventListener('load', function() {
  window.cookieconsent.initialise({
    type: 'opt-in',
    theme: 'classic',
    palette: {
      popup: {
        background: '#dddddd',
        text: '#000000'
      },
      button: {
        background: '#0064cd',
        text: '#ffffff'
      }
    },
    "content": {
      "message": "This site uses cookies to collect anonymous analytics.",
    },
    onInitialise: function () {
      if (this.hasConsented()) {
        enableGtag();
      } else {
        disableGtag();
        deleteCookies(this.options.cookie.name);
      }
    },
    onStatusChange: function() {
      if (this.hasConsented()) {
        enableGtag();
      } else {
        disableGtag();
        deleteCookies(this.options.cookie.name);
      }
    }
  });
});
</script>
`
}
