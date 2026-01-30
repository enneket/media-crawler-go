function rc4_encrypt(plaintext, key) {
    var s = [];
    for (var i = 0; i < 256; i++) {
        s[i] = i;
    }
    var j = 0;
    for (var i = 0; i < 256; i++) {
        j = (j + s[i] + key.charCodeAt(i % key.length)) % 256;
        var temp = s[i];
        s[i] = s[j];
        s[j] = temp;
    }

    var i = 0;
    var j = 0;
    var cipher = [];
    for (var k = 0; k < plaintext.length; k++) {
        i = (i + 1) % 256;
        j = (j + s[i]) % 256;
        var temp = s[i];
        s[i] = s[j];
        s[j] = temp;
        var t = (s[i] + s[j]) % 256;
        cipher.push(String.fromCharCode(s[t] ^ plaintext.charCodeAt(k)));
    }
    return cipher.join('');
}

function le(e, r) {
    return (e << (r %= 32) | e >>> 32 - r) >>> 0
}

function de(e) {
    return 0 <= e && e < 16 ? 2043430169 : 16 <= e && e < 64 ? 2055708042 : void console['error']("invalid j for constant Tj")
}

function pe(e, r, t, n) {
    return 0 <= e && e < 16 ? (r ^ t ^ n) >>> 0 : 16 <= e && e < 64 ? (r & t | r & n | t & n) >>> 0 : (console['error']('invalid j for bool function FF'),
        0)
}

function he(e, r, t, n) {
    return 0 <= e && e < 16 ? (r ^ t ^ n) >>> 0 : 16 <= e && e < 64 ? (r & t | ~r & n) >>> 0 : (console['error']('invalid j for bool function GG'),
        0)
}

function reset() {
    this.reg[0] = 1937774191,
        this.reg[1] = 1226093241,
        this.reg[2] = 388252375,
        this.reg[3] = 3666478592,
        this.reg[4] = 2842636476,
        this.reg[5] = 372324522,
        this.reg[6] = 3817729613,
        this.reg[7] = 2969243214,
        this["chunk"] = [],
        this["size"] = 0
}

function write(e) {
    var a = "string" == typeof e ? function (e) {
        n = encodeURIComponent(e)['replace'](/%([0-9A-F]{2})/g, (function (e, r) {
                return String['fromCharCode']("0x" + r)
            }
        ))
            , a = new Array(n['length']);
        return Array['prototype']['forEach']['call'](n, (function (e, r) {
                a[r] = e.charCodeAt(0)
            }
        )),
            a
    }(e) : e;
    this.size += a.length;
    var f = 64 - this['chunk']['length'];
    if (a['length'] < f)
        this['chunk'] = this['chunk'].concat(a);
    else
        for (this['chunk'] = this['chunk'].concat(a.slice(0, f)); this['chunk'].length >= 64;)
            this['_compress'](this['chunk']),
                f < a['length'] ? this['chunk'] = a['slice'](f, Math['min'](f + 64, a['length'])) : this['chunk'] = [],
                f += 64
}

function sum(e, t) {
    e && (this['reset'](),
        this['write'](e)),
        this['_fill']();
    for (var f = 0; f < this.chunk['length']; f += 64)
        this._compress(this['chunk']['slice'](f, f + 64));
    var i = null;
    if (t == 'hex') {
        i = "";
        for (f = 0; f < 8; f++)
            i += se(this['reg'][f]['toString'](16), 8, "0")
    } else
        for (i = new Array(32),
                 f = 0; f < 8; f++) {
            var c = this.reg[f];
            i[4 * f + 3] = (255 & c) >>> 0,
                c >>>= 8,
                i[4 * f + 2] = (255 & c) >>> 0,
                c >>>= 8,
                i[4 * f + 1] = (255 & c) >>> 0,
                c >>>= 8,
                i[4 * f] = (255 & c) >>> 0
        }
    return this['reset'](),
        i
}

function _compress(t) {
    if (t < 64)
        console.error("compress error: not enough data");
    else {
        for (var f = function (e) {
            for (var r = new Array(132), t = 0; t < 16; t++)
                r[t] = e[4 * t] << 24,
                    r[t] |= e[4 * t + 1] << 16,
                    r[t] |= e[4 * t + 2] << 8,
                    r[t] |= e[4 * t + 3];
            for (t = 16; t < 68; t++) {
                var n = r[t - 16] ^ r[t - 9] ^ le(r[t - 3], 15);
                n = n ^ le(n, 15) ^ le(n, 23),
                    r[t] = (n ^ le(r[t - 13], 7) ^ r[t - 6]) >>> 0
            }
            for (t = 68; t < 132; t++)
                r[t] = (r[t - 68] ^ r[t - 64]) >>> 0;
            return r
        }(t), i = this.reg[0], c = this.reg[1], u = this.reg[2], o = this.reg[3], s = this.reg[4], l = this.reg[5], d = this.reg[6], p = this.reg[7], h = 0; h < 64; h++) {
            var a = (le(i, 12) + s + le(de(h), h)) >>> 0
                , n = (a = le(a, 7)) ^ le(i, 12)
                , e = (pe(h, i, c, u) + o + n + f[h + 68]) >>> 0
                , r = (he(h, s, l, d) + p + a + f[h]) >>> 0;
            o = u,
                u = le(c, 9),
                c = i,
                i = e,
                p = d,
                d = le(l, 19),
                l = s,
                s = (r ^ le(r, 9) ^ le(r, 17)) >>> 0
        }
        this.reg[0] = (this.reg[0] ^ i) >>> 0,
            this.reg[1] = (this.reg[1] ^ c) >>> 0,
            this.reg[2] = (this.reg[2] ^ u) >>> 0,
            this.reg[3] = (this.reg[3] ^ o) >>> 0,
            this.reg[4] = (this.reg[4] ^ s) >>> 0,
            this.reg[5] = (this.reg[5] ^ l) >>> 0,
            this.reg[6] = (this.reg[6] ^ d) >>> 0,
            this.reg[7] = (this.reg[7] ^ p) >>> 0
    }
}

function se(e, r, t, n) {
    for (var a = (e += "")['length']; a < r; a++)
        n ? e += t : e = t + e;
    return e
}

function Sm3() {
    this.reg = new Array(8),
        this.chunk = [],
        this.size = 0,
        this.reset = reset,
        this.write = write,
        this.sum = sum,
        this._compress = _compress,
        this._fill = function () {
            for (var e = 8 * this.size, r = 8 * this.chunk['length'], t = r; t < 64; t++)
                this.chunk[t >> 2] |= 128 << 24 - 8 * (t % 4);
            if (r >= 56) {
                for (t = r; t < 64; t++)
                    this.chunk[t >> 2] |= 0 << 24 - 8 * (t % 4);
                this._compress(this.chunk),
                    this.chunk = []
            }
            for (t = this.chunk.length; t < 14; t++)
                this.chunk[t] = 0;
            this.chunk[14] = Math.floor(e / 4294967296),
                this.chunk[15] = e % 4294967296
        }
        ,
        this.reset()
}

function Utf8ArrayToStr(array) {
    var out, i, len, c;
    var char2, char3;
    out = "";
    len = array.length;
    i = 0;
    while (i < len) {
        c = array[i++];
        switch (c >> 4) {
            case 0:
            case 1:
            case 2:
            case 3:
            case 4:
            case 5:
            case 6:
            case 7:
                out += String.fromCharCode(c);
                break;
            case 12:
            case 13:
                char2 = array[i++];
                out += String.fromCharCode(((c & 0x1F) << 6) | (char2 & 0x3F));
                break;
            case 14:
                char2 = array[i++];
                char3 = array[i++];
                out += String.fromCharCode(((c & 0x0F) << 12) | ((char2 & 0x3F) << 6) | ((char3 & 0x3F) << 0));
                break;
        }
    }
    return out;
}

function result_encrypt(word, key) {
    let sm3 = new Sm3();
    let key_h = sm3.sum(key).slice(0, 16);
    let enc = rc4_encrypt(word, Utf8ArrayToStr(key_h));
    let base64 = btoa(enc);
    return base64
}

function gener_random(num, list) {
    let random_num_list = []
    num = Math.ceil(num)
    for (let i = 0; i < list.length; i++) {
        random_num_list.push(num % list[i]);
        num = Math.floor(num / list[i])
    }
    return random_num_list
}

function generate_rc4_bb_str(url_search_params, user_agent, window_env, cus, arguments) {
    let sm3 = new Sm3();
    let url_search_params_sm3 = sm3.sum(url_search_params, "hex");
    let user_agent_sm3 = sm3.sum(user_agent, "hex");
    let window_env_sm3 = sm3.sum(window_env, "hex");
    let cus_sm3 = sm3.sum(cus, "hex");
    let arguments_sm3 = sm3.sum(JSON.stringify(arguments), "hex");
    let b = new Array(73).fill(0)

    let url_search_params_sm3_list = []
    for (let i = 0; i < url_search_params_sm3.length; i += 2) {
        url_search_params_sm3_list.push(parseInt(url_search_params_sm3.slice(i, i + 2), 16))
    }

    let user_agent_sm3_list = []
    for (let i = 0; i < user_agent_sm3.length; i += 2) {
        user_agent_sm3_list.push(parseInt(user_agent_sm3.slice(i, i + 2), 16))
    }

    let window_env_sm3_list = []
    for (let i = 0; i < window_env_sm3.length; i += 2) {
        window_env_sm3_list.push(parseInt(window_env_sm3.slice(i, i + 2), 16))
    }

    let cus_sm3_list = []
    for (let i = 0; i < cus_sm3.length; i += 2) {
        cus_sm3_list.push(parseInt(cus_sm3.slice(i, i + 2), 16))
    }

    let arguments_sm3_list = []
    for (let i = 0; i < arguments_sm3.length; i += 2) {
        arguments_sm3_list.push(parseInt(arguments_sm3.slice(i, i + 2), 16))
    }

    let cc = url_search_params_sm3_list.slice(0, 16)
    let dd = user_agent_sm3_list.slice(0, 16)
    let ee = window_env_sm3_list.slice(0, 16)
    let ff = cus_sm3_list.slice(0, 16)
    let gg = arguments_sm3_list.slice(0, 16)

    b[0] = 20
    b[1] = 16
    b[2] = cc[14]
    b[3] = cc[15]
    b[4] = cc[7]
    b[5] = cc[8]
    b[6] = cc[9]
    b[7] = cc[10]
    b[8] = cc[11]
    b[9] = cc[12]
    b[10] = cc[13]
    b[11] = cc[6]
    b[12] = cc[5]
    b[13] = cc[4]
    b[14] = cc[3]
    b[15] = cc[2]
    b[16] = cc[1]
    b[17] = cc[0]
    b[18] = dd[0]
    b[19] = dd[1]
    b[20] = dd[2]
    b[21] = dd[3]
    b[22] = dd[4]
    b[23] = dd[5]
    b[24] = dd[6]
    b[25] = dd[7]
    b[26] = dd[8]
    b[27] = dd[9]
    b[28] = dd[10]
    b[29] = dd[11]
    b[30] = dd[12]
    b[31] = dd[13]
    b[32] = dd[14]
    b[33] = dd[15]
    b[34] = ee[0]
    b[35] = ee[1]
    b[36] = ee[2]
    b[37] = ee[3]
    b[38] = ee[4]
    b[39] = ee[5]
    b[40] = ee[6]
    b[41] = ee[7]
    b[42] = ee[8]
    b[43] = ee[9]
    b[44] = ee[10]
    b[45] = ee[11]
    b[46] = ee[12]
    b[47] = ee[13]
    b[48] = ee[14]
    b[49] = ee[15]
    b[50] = ff[0]
    b[51] = ff[1]
    b[52] = ff[2]
    b[53] = ff[3]
    b[54] = ff[4]
    b[55] = ff[5]
    b[56] = ff[6]
    b[57] = ff[7]
    b[58] = ff[8]
    b[59] = ff[9]
    b[60] = ff[10]
    b[61] = ff[11]
    b[62] = ff[12]
    b[63] = ff[13]
    b[64] = ff[14]
    b[65] = ff[15]
    b[66] = gg[0]
    b[67] = gg[1]
    b[68] = gg[2]
    b[69] = gg[3]
    b[70] = gg[4]
    b[71] = gg[5]
    b[72] = gg[6]
    let window_env_list = window_env.split("|").map(function (x) {
        return parseInt(x)
    })
    bb = [
        b[12], b[13], b[14], b[15], b[16], b[17], b[11], b[10], b[9], b[8], b[7], b[6], b[5], b[4],
        b[3], b[2], b[59], b[46], b[47], b[48], b[49], b[50], b[24], b[25], b[65], b[66], b[70], b[71]
    ]
    bb = bb.concat(window_env_list).concat(b[72])
    return rc4_encrypt(String.fromCharCode.apply(null, bb), String.fromCharCode.apply(null, [121]));
}

function generate_random_str() {
    let random_str_list = []
    random_str_list = random_str_list.concat(gener_random(Math.random() * 10000, [3, 45]))
    random_str_list = random_str_list.concat(gener_random(Math.random() * 10000, [1, 0]))
    random_str_list = random_str_list.concat(gener_random(Math.random() * 10000, [1, 5]))
    return String.fromCharCode.apply(null, random_str_list)
}

function sign(url_search_params, user_agent, arguments) {
    let result_str = generate_random_str() + generate_rc4_bb_str(
        url_search_params,
        user_agent,
        "1536|747|1536|834|0|30|0|0|1536|834|1536|864|1525|747|24|24|Win32",
        "cus",
        arguments
    );
    return result_encrypt(result_str, "s4") + "=";
}

function sign_datail(params, userAgent) {
    return sign(params, userAgent, [0, 1, 14])
}

function sign_reply(params, userAgent) {
    return sign(params, userAgent, [0, 1, 8])
}
