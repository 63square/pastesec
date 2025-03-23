const std = @import("std");
const xchacha = std.crypto.aead.chacha_poly.XChaCha20Poly1305;
const argon2 = std.crypto.pwhash.argon2;

const tag_length = xchacha.tag_length;
const key_length = xchacha.key_length;
const nonce_length = xchacha.nonce_length;

const allocator = std.heap.wasm_allocator;

export fn deriveKey(s: [*]u8, length: usize) i32 {
    const salt = s[0..nonce_length];

    var derived_key: [key_length]u8 = undefined;
    argon2.kdf(
        allocator,
        &derived_key,
        s[nonce_length .. length + nonce_length],
        salt,
        argon2.Params.owasp_2id,
        .argon2id,
    ) catch {
        return -1;
    };

    @memcpy(s[0..key_length], &derived_key);
    return key_length;
}

export fn encrypt(s: [*]u8, length: usize) i32 {
    const total_len: usize = length + key_length + nonce_length * 2;

    var key: [key_length]u8 = undefined;
    @memcpy(&key, s[0..key_length]);

    const new_len = total_len - (key_length - tag_length);
    @memcpy(s[tag_length..new_len], s[key_length..total_len]);

    xchacha.encrypt(
        s[tag_length + nonce_length * 2 .. new_len],
        s[0..tag_length],
        s[tag_length + nonce_length * 2 .. new_len],
        s[tag_length .. tag_length + nonce_length],
        s[tag_length + nonce_length .. tag_length + nonce_length * 2].*,
        key,
    );

    return @intCast(new_len);
}

export fn decrypt(s: [*]u8, length: usize) i32 {
    const total_len: usize = length - (tag_length + nonce_length * 2);

    const c = s[key_length .. key_length + length];

    const tag = c[0..tag_length];
    const salt = c[tag_length .. tag_length + nonce_length];
    const nonce = c[tag_length + nonce_length .. tag_length + nonce_length * 2];
    const ciphertext = c[tag_length + nonce_length * 2 ..];

    xchacha.decrypt(
        s[0..total_len],
        ciphertext,
        tag.*,
        salt,
        nonce.*,
        s[0..key_length].*,
    ) catch {
        return -1;
    };

    return @intCast(total_len);
}
