-- name: CreateDelayProfile :one
INSERT INTO delay_profiles (
    enable_usenet, enable_torrent, preferred_protocol,
    usenet_delay, torrent_delay, sort_order,
    bypass_if_highest_quality, bypass_if_above_custom_format_score,
    minimum_custom_format_score, tags
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, enable_usenet, enable_torrent, preferred_protocol,
          usenet_delay, torrent_delay, sort_order,
          bypass_if_highest_quality, bypass_if_above_custom_format_score,
          minimum_custom_format_score, tags;

-- name: GetDelayProfileByID :one
SELECT id, enable_usenet, enable_torrent, preferred_protocol,
       usenet_delay, torrent_delay, sort_order,
       bypass_if_highest_quality, bypass_if_above_custom_format_score,
       minimum_custom_format_score, tags
FROM delay_profiles WHERE id = $1;

-- name: ListDelayProfiles :many
SELECT id, enable_usenet, enable_torrent, preferred_protocol,
       usenet_delay, torrent_delay, sort_order,
       bypass_if_highest_quality, bypass_if_above_custom_format_score,
       minimum_custom_format_score, tags
FROM delay_profiles ORDER BY sort_order, id;

-- name: UpdateDelayProfile :exec
UPDATE delay_profiles
SET enable_usenet = $1, enable_torrent = $2, preferred_protocol = $3,
    usenet_delay = $4, torrent_delay = $5, sort_order = $6,
    bypass_if_highest_quality = $7, bypass_if_above_custom_format_score = $8,
    minimum_custom_format_score = $9, tags = $10
WHERE id = $11;

-- name: DeleteDelayProfile :exec
DELETE FROM delay_profiles WHERE id = $1;
