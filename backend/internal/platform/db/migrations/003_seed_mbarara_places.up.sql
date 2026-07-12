INSERT INTO places (city_id, name, landmark_note, location, category)
SELECT c.id, p.name, p.landmark, ST_SetSRID(ST_MakePoint(p.lng, p.lat), 4326)::geography, p.category
FROM cities c
CROSS JOIN (VALUES
  ('Mbarara University Main Gate', 'Main campus entrance', 30.6530, -0.6085, 'campus'),
  ('Mbarara Regional Referral Hospital', 'Hospital main entrance', 30.6490, -0.6010, 'hospital'),
  ('Mbarara Bus Park', 'Central bus terminal', 30.6580, -0.6040, 'transport'),
  ('Clock Tower', 'Town centre landmark', 30.6586, -0.6072, 'landmark'),
  ('Lake View Hotel', 'Popular town hotel', 30.6610, -0.6090, 'hotel')
) AS p(name, landmark, lng, lat, category)
WHERE c.slug = 'mbarara';
