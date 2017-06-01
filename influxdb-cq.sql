

# byCountry agg 1h, 4h, 12h 24h
CREATE CONTINUOUS QUERY "requests_1h" ON "catalog" BEGIN SELECT count(distinct("ip")) AS "unique", count("ip") AS "total" INTO "requests_1h" FROM "requests" GROUP BY time(1h),country,path,city,ip END

CREATE CONTINUOUS QUERY "requests_4h" ON "catalog" BEGIN SELECT last("total") AS "total", last("unique") AS "unique" INTO "requests_4h" FROM "requests_1h" GROUP BY time(4h),country,path,city,ip END

CREATE CONTINUOUS QUERY "requests_12h" ON "catalog" BEGIN SELECT last("total") AS "total", last("unique") AS "unique" INTO "requests_12h" FROM "requests_4h" GROUP BY time(12h),country,path,city,ip END

CREATE CONTINUOUS QUERY "requests_24h" ON "catalog" BEGIN SELECT last("total") AS "total", last("unique") AS "unique" INTO "requests_24h" FROM "requests_12h" GROUP BY time(24h),country,path,city,ip END

# byCountry agg 1h, 4h, 12h 24h
CREATE CONTINUOUS QUERY "byCountry_1h" ON "catalog" BEGIN SELECT count(distinct("ip")) AS "unique", count("ip") AS "total" INTO "byCountry_1h" FROM "requests" GROUP BY time(1h),country,country_isocode,path,city END

CREATE CONTINUOUS QUERY "byCountry_4h" ON "catalog" BEGIN SELECT last("total") AS "total", last("unique") AS "unique" INTO "byCountry_4h" FROM "byCountry_1h" GROUP BY time(4h),country,country_isocode,path,city END

CREATE CONTINUOUS QUERY "byCountry_12h" ON "catalog" BEGIN SELECT last("total") AS "total", last("unique") AS "unique" INTO "byCountry_12h" FROM "byCountry_4h" GROUP BY time(12h),country,country_isocode,path,city,ip END

CREATE CONTINUOUS QUERY "byCountry_24h" ON "catalog" BEGIN SELECT last("total") AS "total", last("unique") AS "unique" INTO "byCountry_24h" FROM "byCountry_12h" GROUP BY time(24h),country,country_isocode,path,city,ip END
