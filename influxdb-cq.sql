# byIp agg 24h
CREATE CONTINUOUS QUERY "byIp_24h" ON "catalog" RESAMPLE EVERY 6h BEGIN SELECT distinct("ip") AS ip INTO "byIp_24h" FROM "requests" WHERE "uid" != '-' GROUP BY time(24h),path,ip END

# byUid_history agg 24h
CREATE CONTINUOUS QUERY "byUid_history_24h" ON "catalog" RESAMPLE EVERY 6h BEGIN SELECT distinct("uid") AS uid INTO "byUid_history_24h" FROM "requests" GROUP BY time(24h),path,uid END

# byUid agg 24h
CREATE CONTINUOUS QUERY "byUid_1h" ON "catalog" BEGIN SELECT count(distinct("uid")) AS "unique", count(distinct("ip")) AS "unique_ip", last("uid") AS "uid" INTO "byUid_1h" FROM "requests" GROUP BY time(1h),uid,path END
CREATE CONTINUOUS QUERY "byUid_24h" ON "catalog" BEGIN SELECT last("unique") AS "unique", last("unique_ip") AS "unique_ip", last("uid") AS "uid" INTO "byUid_24h" FROM "byUid_1h" GROUP BY time(24h),uid,path END

SELECT count(distinct("uid")) AS "unique", count(distinct("ip")) AS "unique_ip", last("uid") AS "uid" INTO "byUid2_24h" FROM "requests" where time >= '2018-10-13' AND time < '2018-10-14' GROUP BY time(24h),path

# byCountry agg 24h
CREATE CONTINUOUS QUERY "byCountry_1h" ON "catalog" BEGIN SELECT count("ip") AS total, count(distinct("ip")) AS unique INTO "byCountry_1h" FROM "requests" GROUP BY time(1h), country, country_isocode, path END
CREATE CONTINUOUS QUERY "byCountry_24h" ON "catalog" BEGIN SELECT last("total") AS total, last("unique") AS unique INTO "byCountry_24h" FROM "byCountry_1h" GROUP BY time(1d), country, country_isocode, path END

SELECT distinct(ip) AS ip FROM catalog.autogen.requests WHERE uid != '-' AND time >= '2018-10-16T00:00:00Z' AND time < '2018-10-17T00:00:00Z' GROUP BY time(1d), path, ip
SELECT distinct(uid) AS uid  FROM catalog.autogen.requests WHERE time >= '2018-10-16T00:00:00Z' AND time < '2018-10-17T00:00:00Z' GROUP BY time(1d), path, uid


SELECT distinct(uid) AS uid INTO catalog.autogen.byUid_history_24h FROM catalog.autogen.requests WHERE time >= '2018-10-17T00:00:00Z' AND time < '2018-10-18T00:00:00Z' GROUP BY time(1d), path, uid