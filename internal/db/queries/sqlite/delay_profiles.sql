-- name: CreateDelayProfile :one
INSERT INTO delay_profiles (
    enable_usenet, enable_torrent, preferred_protocol,
    usenet_delay, torrent_delay, sort_order,
    bypass_if_highest_quality, bypass_if_above_custom_format_score,
    minimum_custom_format_score, tags
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, enable_usenet, enable_torrent, preferred_protocol,
          usenet_delay, torrent_delay, sort_order,
          bypass_if_highest_quality, bypass_if_above_custom_format_score,
          minimum_custom_format_score, tags;

-- name: GetDelayProfileByID :one
SELECT id, enable_usenet, enable_torrent, preferred_protocol,
       usenet_delay, torrent_delay, sort_order,
       bypass_if_highest_quality, bypass_if_above_custom_format_score,
       minimum_custom_format_score, tags
FROM delay_profiles WHERE id = ?;

-- name: ListDelayProfiles :many
SELECT id, enable_usenet, enable_torrent, preferred_protocol,
       usenet_delay, torrent_delay, sort_order,
       bypass_if_highest_quality, bypass_if_above_custom_format_score,
       minimum_custom_format_score, tags
FROM delay_profiles ORDER BY sort_order, id;

-- name: UpdateDelayProfile :exec
UPDATE delay_profiles
SET enable_usenet = ?, enable_torrent = ?, preferred_protocol = ?,
    usenet_delay = ?, torrent_delay = ?, sort_order = ?,
    bypass_if_highest_quality = ?, bypass_if_above_custom_format_score = ?,
    minimum_custom_format_score = ?, tags = ?
WHERE id = ?;

-- name: DeleteDelayProfile :exec
DELETE FROM delay_profiles WHERE id = ?;
