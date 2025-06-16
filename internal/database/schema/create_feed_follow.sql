WITHinsAS(
  INSERT INTO feed_follows(
    user_id,
    feed_id
  )
VALUES(
  ?,
  ?
) RETURNINGid,
created_at,
updated_at,
user_id,
feed_id
)SELECT
  ins.id,
  ins.created_at,
  ins.updated_at,
  ins.user_id,
  ins.feed_id,
  u.name AS user_name,
  f.name AS feed_name,
  f.url AS feed_url
FROM
  ins
JOIN users AS u
  ON u.id = ins.user_id
JOIN feeds AS f
  ON f.id = ins.feed_id;
