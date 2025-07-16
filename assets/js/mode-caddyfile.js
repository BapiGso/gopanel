ace.define("ace/mode/caddyfile_highlight_rules", ["require", "exports", "module", "ace/lib/oop", "ace/mode/text_highlight_rules"], function(require, exports, module) {
    "use strict";

    var oop = require("../lib/oop");
    var TextHighlightRules = require("./text_highlight_rules").TextHighlightRules;

    var CaddyfileHighlightRules = function() {

        var keywords = ( // Common sub-directives or keywords appearing within blocks or as arguments
            "ask|try_files|try_policy|split_path|ca|crt|key|output|file|format|level|protocols|" +
            "client_auth|mode|ciphers|curves|alpn|fallback|get_certificate|ignore_loaded|policy|" +
            "insecure_skip_verify|issuer|leaf|load|must_staple|on_demand|preferred_chains|subjects|" +
            "with|method|path|path_regexp|query|header_field|not|expression|host|source_ip|protocol|" +
            "mismatch|status|strict|to|code|body|close|latency|transport|split|dial_timeout|health_uri|" +
            "health_interval|health_timeout|lb_policy|lb_try_duration|lb_try_interval|fail_duration|" +
            "max_fails|max_requests|unhealthy_status|unhealthy_latency|upstreams|except|args|" +
            "strip_path_prefix|strip_path_suffix|uri_substring|status_code|headers|cookies|resolvers|" +
            "min_files|max_files|include_acme|order|max_size|roll_disabled|roll_keep|roll_keep_for|" +
            "roll_local_time|allow|deny" // Added from Chroma output
        );

        var directives = ( // Main Caddy directives
            "abort|acme_server|admin|bind|basic_auth|log|encode|errors|file_server|forward_auth|" +
            "handle|handle_errors|handle_path|header|import|invoke|map|metrics|php_fastcgi|push|" +
            "redir|request_body|request_header|respond|reverse_proxy|rewrite|root|route|buffer|" +
            "templates|tls|try_files|uri|vars|webdav|acme_dns|authentication|authorize|cache|cbor|" +
            "circuit_breaker|clamav|conn_limit|copy|debug|dns|docker_proxy|dyndns|early_hints|events|" +
            "exec|expvar|fastcgi|forward_proxy|geoip|git|grpc_proxy|grpc_web|health_check|http_cache|" +
            "ip_range|jarm|jwt|ldap|limits|load_balance|local_acme|mercure|method|minify|mock|mtls|" +
            "multipass|mutex|net|ntlm|paseto|password|path_escapes|pprof|prometheus|proxyprotocol|" +
            "ratelimit|replace|resolve|retry|split|sql|ssh|static_response|supervisor|syslog|tee|" +
            "timeout|tracing|transform|tunnel|udp|untar|unzip|upload|user|virtual_host|waf|xcaddy|" +
            "xml|xsrf|yaml|zip|zstd|on_demand_tls" // Added from Chroma: on_demand_tls, debug, admin (handled as global)
                                                   // redir is present.
        );

        var globalOptions = ( // Options that can appear in a global block {}
            "debug|admin|log_credentials|http_port|https_port|default_sni|local_certs|storage_clean_interval|" +
            "ocsp_stapling|ocsp_strict|on_demand_tls|persist_config|skip_install_trust|storage|" +
            "events|order|pki|servers|auto_https|acme_ca|acme_eab|email|cert_issuer|renewal_window_ratio|" +
            "key_type|resolvers|trusted_proxies" // Includes debug, admin, on_demand_tls
        );

        var constants = ( // true, false, on, off
            "on|off|true|false"
        );

        this.$rules = {
            "start": [
                { token: "comment.line.number-sign.caddyfile", regex: /#.*/ },
                { token: "string.quoted.double.caddyfile", regex: /"/, next: "qqstring" },

                // Snippet definition: (name)
                { token: "entity.name.tag.snippet.definition.caddyfile", regex: /\([\w.-]+\)/ },

                // Global options block at the very beginning
                {
                    token: "punctuation.definition.block.begin.caddyfile",
                    regex: /^\s*{/, // Matches { at the beginning of a line, possibly with whitespace
                    push: "global_block",
                    next : "global_block_start_line" // Transition to handle directives on the same line
                },

                // Site Addresses: example.com, *.example.com, :8080, localhost, etc.
                // Followed by { or appearing alone or comma-separated
                {
                    token: [
                        "entity.name.tag.site_address.caddyfile",
                        "text", // Whitespace
                        "punctuation.separator.comma.caddyfile", // Comma
                        "text" // Whitespace
                    ],
                    regex: /([^#@{\s,\(][^\s{,\(]*)([ \t]*)(,)([ \t]*)/, // site, ws, comma, ws
                    next: "site_address_maybe_comma"
                },
                { // Last site address in a comma list, or a single one, followed by {
                    token: [
                        "entity.name.tag.site_address.caddyfile",
                        "text", // Whitespace
                        "punctuation.definition.block.begin.caddyfile"
                    ],
                    regex: /([^#@{\s,\(][^\s{,\(]*)([ \t]*)({)/,
                    push: "block_content"
                },
                { // Site address on a line by itself (directives typically follow)
                    token: "entity.name.tag.site_address.caddyfile",
                    regex: /^[^#@{\s,\(][^\s{,\(]*$/,
                    // Next line will be parsed in "start" or we might need a state if indentation matters strictly
                },


                // Matcher definition: @name
                {
                    token: "entity.name.tag.matcher.definition.caddyfile",
                    regex: /@[a-zA-Z_][\w-]*/
                },
                // Star matcher (often used with directives)
                {
                    token: "keyword.operator.matcher.caddyfile",
                    regex: /\*/
                },

                // Placeholders: {vars.my_var}, {http.request.host}
                { token: "variable.other.placeholder.caddyfile", regex: /{[^}]+}/ },

                { token: "keyword.control.directive.caddyfile", regex: "\\b(?:" + directives + ")\\b" },
                { token: "keyword.other.caddyfile", regex: "\\b(?:" + keywords + ")\\b" },
                { token: "constant.language.caddyfile", regex: "\\b(?:" + constants + ")\\b" },
                { token: "constant.numeric.caddyfile", regex: /\b\d+\b/ },

                // Open brace for a generic block (if not caught by site address or global)
                { token: "punctuation.definition.block.begin.caddyfile", regex: /{/, push: "block_content" },
                // Close brace for a generic block
                { token: "punctuation.definition.block.end.caddyfile", regex: /}/, next: "pop" } // Should pop from block_content or global_block
            ],

            "site_address_maybe_comma": [ // After "site," expecting more site addresses or "{"
                {
                    token: [
                        "entity.name.tag.site_address.caddyfile",
                        "text", // Whitespace
                        "punctuation.separator.comma.caddyfile", // Comma
                        "text" // Whitespace
                    ],
                    regex: /([^#@{\s,\(][^\s{,\(]*)([ \t]*)(,)([ \t]*)/, // site, ws, comma, ws (loop)
                },
                { // Last site address in a comma list, or a single one, followed by {
                    token: [
                        "entity.name.tag.site_address.caddyfile",
                        "text", // Whitespace
                        "punctuation.definition.block.begin.caddyfile"
                    ],
                    regex: /([^#@{\s,\(][^\s{,\(]*)([ \t]*)({)/,
                    next: "pop", // Pop this state
                    push: "block_content"
                },
                { // Site address on a line by itself (directives typically follow)
                    token: "entity.name.tag.site_address.caddyfile",
                    regex: /[^#@{\s,\(][^\s{,\(]*$/,
                    next: "pop" // Pop this state
                },
                {defaultToken: "text", next: "pop"} // Fallback and pop
            ],

            "global_block_start_line": [ // Immediately after global { to parse rest of the line
                { token: "comment.line.number-sign.caddyfile", regex: /#.*/, next: "pop" }, // Comment ends the line processing
                { token: "string.quoted.double.caddyfile", regex: /"/, next: "qqstring_global_block_pop" }, // String in global, then pop to global_block
                { token: "keyword.control.global.caddyfile", regex: "\\b(?:" + globalOptions + ")\\b" },
                { token: "keyword.other.caddyfile", regex: "\\b(?:" + keywords + ")\\b" }, // Some general keywords might be here
                { token: "constant.language.caddyfile", regex: "\\b(?:" + constants + ")\\b" },
                { token: "constant.numeric.caddyfile", regex: /\b\d+\b/ },
                { token: "variable.other.placeholder.caddyfile", regex: /{[^}]+}/ },
                { token: "punctuation.definition.block.begin.caddyfile", regex: /{/, push: "global_block" }, // Nested global block
                { token: "punctuation.definition.block.end.caddyfile", regex: /}/, next: "pop" }, // End of global block
                { defaultToken: "text", next: "pop" } // If anything else, pop to global_block for next line
            ],

            "global_block": [ // Inside a global options block {}
                { token: "comment.line.number-sign.caddyfile", regex: /#.*/ },
                { token: "string.quoted.double.caddyfile", regex: /"/, next: "qqstring_global_block" },
                { token: "keyword.control.global.caddyfile", regex: "\\b(?:" + globalOptions + ")\\b" },
                { token: "keyword.other.caddyfile", regex: "\\b(?:" + keywords + ")\\b" },
                { token: "constant.language.caddyfile", regex: "\\b(?:" + constants + ")\\b" },
                { token: "constant.numeric.caddyfile", regex: /\b\d+\b/ },
                { token: "variable.other.placeholder.caddyfile", regex: /{[^}]+}/ },
                // Matcher definition: @name (can appear in global block for snippet-like global options)
                {
                    token: "entity.name.tag.matcher.definition.caddyfile",
                    regex: /@[a-zA-Z_][\w-]*/
                },
                { token: "punctuation.definition.block.begin.caddyfile", regex: /{/, push: "global_block" }, // Nested global block
                { token: "punctuation.definition.block.end.caddyfile", regex: /}/, next: "pop" },
                { defaultToken: "text" }
            ],

            "block_content": [ // Inside a directive block {} or site block {}
                { token: "comment.line.number-sign.caddyfile", regex: /#.*/ },
                { token: "string.quoted.double.caddyfile", regex: /"/, next: "qqstring_block" },

                // Snippet definition: (name)
                { token: "entity.name.tag.snippet.definition.caddyfile", regex: /\([\w.-]+\)/ },

                // Matcher definition or usage: @name
                {
                    token: "entity.name.tag.matcher.caddyfile", // Generic for definition or usage inside block
                    regex: /@[a-zA-Z_][\w-]*/
                },
                // Star matcher
                {
                    token: "keyword.operator.matcher.caddyfile",
                    regex: /\*/
                },

                { token: "variable.other.placeholder.caddyfile", regex: /{[^}]+}/ },
                // For 'import snippet_name'
                {
                    token: ["keyword.control.directive.caddyfile", "text", "variable.parameter.snippet.caddyfile"],
                    regex: "\\b(import)(\\s+)([a-zA-Z_][\\w-]*)\\b"
                },
                { token: "keyword.control.directive.caddyfile", regex: "\\b(?:" + directives + ")\\b" },
                { token: "keyword.other.caddyfile", regex: "\\b(?:" + keywords + ")\\b" },
                { token: "constant.language.caddyfile", regex: "\\b(?:" + constants + ")\\b" },
                { token: "constant.numeric.caddyfile", regex: /\b\d+\b/ },
                { token: "punctuation.definition.block.begin.caddyfile", regex: /{/, push: "block_content" }, // Nested block
                { token: "punctuation.definition.block.end.caddyfile", regex: /}/, next: "pop" },
                { defaultToken: "text" }
            ],

            "qqstring": [ // Standard quoted string in "start" state
                { token: "constant.character.escape.caddyfile", regex: /\\./ },
                { token: "string.quoted.double.caddyfile", regex: /"/, next: "start" },
                { defaultToken: "string.quoted.double.caddyfile" }
            ],
            "qqstring_global_block": [ // Quoted string in "global_block" state
                { token: "constant.character.escape.caddyfile", regex: /\\./ },
                { token: "string.quoted.double.caddyfile", regex: /"/, next: "global_block" },
                { defaultToken: "string.quoted.double.caddyfile" }
            ],
            "qqstring_global_block_pop": [ // Quoted string in "global_block_start_line" state (pops to global_block)
                { token: "constant.character.escape.caddyfile", regex: /\\./ },
                { token: "string.quoted.double.caddyfile", regex: /"/, next: "global_block" }, // after string, go to global_block for next line
                { defaultToken: "string.quoted.double.caddyfile" }
            ],
            "qqstring_block": [ // Quoted string in "block_content" state
                { token: "constant.character.escape.caddyfile", regex: /\\./ },
                { token: "string.quoted.double.caddyfile", regex: /"/, next: "block_content" },
                { defaultToken: "string.quoted.double.caddyfile" }
            ]
        };

        this.normalizeRules();
    };

    oop.inherits(CaddyfileHighlightRules, TextHighlightRules);

    exports.CaddyfileHighlightRules = CaddyfileHighlightRules;
});

ace.define("ace/mode/caddyfile", ["require", "exports", "module", "ace/lib/oop", "ace/mode/text", "ace/mode/caddyfile_highlight_rules", "ace/mode/folding/cstyle"], function(require, exports, module) {
    "use strict";

    var oop = require("../lib/oop");
    var TextMode = require("./text").Mode;
    var CaddyfileHighlightRules = require("./caddyfile_highlight_rules").CaddyfileHighlightRules;
    var CStyleFoldMode = require("./folding/cstyle").FoldMode; // C-style folding for { }

    var Mode = function() {
        this.HighlightRules = CaddyfileHighlightRules;
        this.foldingRules = new CStyleFoldMode();
        this.$behaviour = this.$defaultBehaviour;
    };
    oop.inherits(Mode, TextMode);

    (function() {
        this.lineCommentStart = "#";
        // Auto-pairing of braces, quotes, etc. can be configured here if needed
        // this.$quotes = { "{": "}", '"': '"', "'": "'" }; // Example

        this.$id = "ace/mode/caddyfile";
    }).call(Mode.prototype);

    exports.Mode = Mode;
});