# Audiobookshelf API Inventory

- Source ref: `2d0a5462d2234a8c1f853c9c23b790dc8e690fb5`
- Source path: `server/routers/ApiRouter.js`
- Total routes: `198`
- Read-only routes: `83`
- Mutating routes: `115`

## Routes

| Method | Path | Handler | Middleware | Mutates |
| --- | --- | --- | --- | --- |
| `GET` | `/api-keys` | `ApiKeyController.getAll` | `ApiKeyController.middleware` | `false` |
| `POST` | `/api-keys` | `ApiKeyController.create` | `ApiKeyController.middleware` | `true` |
| `DELETE` | `/api-keys/:id` | `ApiKeyController.delete` | `ApiKeyController.middleware` | `true` |
| `PATCH` | `/api-keys/:id` | `ApiKeyController.update` | `ApiKeyController.middleware` | `true` |
| `GET` | `/auth-settings` | `MiscController.getAuthSettings` |  | `false` |
| `PATCH` | `/auth-settings` | `MiscController.updateAuthSettings` |  | `true` |
| `POST` | `/authorize` | `MiscController.authorize` |  | `true` |
| `DELETE` | `/authors/:id` | `AuthorController.delete` | `AuthorController.middleware` | `true` |
| `GET` | `/authors/:id` | `AuthorController.findOne` | `AuthorController.middleware` | `false` |
| `PATCH` | `/authors/:id` | `AuthorController.update` | `AuthorController.middleware` | `true` |
| `DELETE` | `/authors/:id/image` | `AuthorController.deleteImage` | `AuthorController.middleware` | `true` |
| `GET` | `/authors/:id/image` | `AuthorController.getImage` |  | `false` |
| `POST` | `/authors/:id/image` | `AuthorController.uploadImage` | `AuthorController.middleware` | `true` |
| `POST` | `/authors/:id/match` | `AuthorController.match` | `AuthorController.middleware` | `true` |
| `GET` | `/backups` | `BackupController.getAll` | `BackupController.middleware` | `false` |
| `POST` | `/backups` | `BackupController.create` | `BackupController.middleware` | `true` |
| `DELETE` | `/backups/:id` | `BackupController.delete` | `BackupController.middleware` | `true` |
| `GET` | `/backups/:id/apply` | `BackupController.apply` | `BackupController.middleware` | `false` |
| `GET` | `/backups/:id/download` | `BackupController.download` | `BackupController.middleware` | `false` |
| `PATCH` | `/backups/path` | `BackupController.updatePath` | `BackupController.middleware` | `true` |
| `POST` | `/backups/upload` | `BackupController.upload` | `BackupController.middleware` | `true` |
| `POST` | `/cache/items/purge` | `CacheController.purgeItemsCache` |  | `true` |
| `POST` | `/cache/purge` | `CacheController.purgeCache` |  | `true` |
| `GET` | `/collections` | `CollectionController.findAll` |  | `false` |
| `POST` | `/collections` | `CollectionController.create` | `CollectionController.middleware` | `true` |
| `DELETE` | `/collections/:id` | `CollectionController.delete` | `CollectionController.middleware` | `true` |
| `GET` | `/collections/:id` | `CollectionController.findOne` | `CollectionController.middleware` | `false` |
| `PATCH` | `/collections/:id` | `CollectionController.update` | `CollectionController.middleware` | `true` |
| `POST` | `/collections/:id/batch/add` | `CollectionController.addBatch` | `CollectionController.middleware` | `true` |
| `POST` | `/collections/:id/batch/remove` | `CollectionController.removeBatch` | `CollectionController.middleware` | `true` |
| `POST` | `/collections/:id/book` | `CollectionController.addBook` | `CollectionController.middleware` | `true` |
| `DELETE` | `/collections/:id/book/:bookId` | `CollectionController.removeBook` | `CollectionController.middleware` | `true` |
| `GET` | `/custom-metadata-providers` | `CustomMetadataProviderController.getAll` | `CustomMetadataProviderController.middleware` | `false` |
| `POST` | `/custom-metadata-providers` | `CustomMetadataProviderController.create` | `CustomMetadataProviderController.middleware` | `true` |
| `DELETE` | `/custom-metadata-providers/:id` | `CustomMetadataProviderController.delete` | `CustomMetadataProviderController.middleware` | `true` |
| `POST` | `/emails/ereader-devices` | `EmailController.updateEReaderDevices` | `EmailController.adminMiddleware` | `true` |
| `POST` | `/emails/send-ebook-to-device` | `EmailController.sendEBookToDevice` |  | `true` |
| `GET` | `/emails/settings` | `EmailController.getSettings` | `EmailController.adminMiddleware` | `false` |
| `PATCH` | `/emails/settings` | `EmailController.updateSettings` | `EmailController.adminMiddleware` | `true` |
| `POST` | `/emails/test` | `EmailController.sendTest` | `EmailController.adminMiddleware` | `true` |
| `GET` | `/feeds` | `RSSFeedController.getAll` | `RSSFeedController.middleware` | `false` |
| `POST` | `/feeds/:id/close` | `RSSFeedController.closeRSSFeed` | `RSSFeedController.middleware` | `true` |
| `POST` | `/feeds/collection/:collectionId/open` | `RSSFeedController.openRSSFeedForCollection` | `RSSFeedController.middleware` | `true` |
| `POST` | `/feeds/item/:itemId/open` | `RSSFeedController.openRSSFeedForItem` | `RSSFeedController.middleware` | `true` |
| `POST` | `/feeds/series/:seriesId/open` | `RSSFeedController.openRSSFeedForSeries` | `RSSFeedController.middleware` | `true` |
| `GET` | `/filesystem` | `FileSystemController.getPaths` |  | `false` |
| `POST` | `/filesystem/pathexists` | `FileSystemController.checkPathExists` |  | `true` |
| `GET` | `/genres` | `MiscController.getAllGenres` |  | `false` |
| `DELETE` | `/genres/:genre` | `MiscController.deleteGenre` |  | `true` |
| `POST` | `/genres/rename` | `MiscController.renameGenre` |  | `true` |
| `DELETE` | `/items/:id` | `LibraryItemController.delete` | `LibraryItemController.middleware` | `true` |
| `GET` | `/items/:id` | `LibraryItemController.findOne` | `LibraryItemController.middleware` | `false` |
| `POST` | `/items/:id/chapters` | `LibraryItemController.updateMediaChapters` | `LibraryItemController.middleware` | `true` |
| `DELETE` | `/items/:id/cover` | `LibraryItemController.removeCover` | `LibraryItemController.middleware` | `true` |
| `GET` | `/items/:id/cover` | `LibraryItemController.getCover` |  | `false` |
| `PATCH` | `/items/:id/cover` | `LibraryItemController.updateCover` | `LibraryItemController.middleware` | `true` |
| `POST` | `/items/:id/cover` | `LibraryItemController.uploadCover` | `LibraryItemController.middleware` | `true` |
| `GET` | `/items/:id/download` | `LibraryItemController.download` | `LibraryItemController.middleware` | `false` |
| `PATCH` | `/items/:id/ebook/:fileid/status` | `LibraryItemController.updateEbookFileStatus` | `LibraryItemController.middleware` | `true` |
| `GET` | `/items/:id/ebook/:fileid?` | `LibraryItemController.getEBookFile` | `LibraryItemController.middleware` | `false` |
| `GET` | `/items/:id/ffprobe/:fileid` | `LibraryItemController.getFFprobeData` | `LibraryItemController.middleware` | `false` |
| `DELETE` | `/items/:id/file/:fileid` | `LibraryItemController.deleteLibraryFile` | `LibraryItemController.middleware` | `true` |
| `GET` | `/items/:id/file/:fileid` | `LibraryItemController.getLibraryFile` | `LibraryItemController.middleware` | `false` |
| `GET` | `/items/:id/file/:fileid/download` | `LibraryItemController.downloadLibraryFile` | `LibraryItemController.middleware` | `false` |
| `POST` | `/items/:id/match` | `LibraryItemController.match` | `LibraryItemController.middleware` | `true` |
| `PATCH` | `/items/:id/media` | `LibraryItemController.updateMedia` | `LibraryItemController.middleware` | `true` |
| `GET` | `/items/:id/metadata-object` | `LibraryItemController.getMetadataObject` | `LibraryItemController.middleware` | `false` |
| `POST` | `/items/:id/play` | `LibraryItemController.startPlaybackSession` | `LibraryItemController.middleware` | `true` |
| `POST` | `/items/:id/play/:episodeId` | `LibraryItemController.startEpisodePlaybackSession` | `LibraryItemController.middleware` | `true` |
| `POST` | `/items/:id/scan` | `LibraryItemController.scan` | `LibraryItemController.middleware` | `true` |
| `PATCH` | `/items/:id/tracks` | `LibraryItemController.updateTracks` | `LibraryItemController.middleware` | `true` |
| `POST` | `/items/batch/delete` | `LibraryItemController.batchDelete` |  | `true` |
| `POST` | `/items/batch/get` | `LibraryItemController.batchGet` |  | `true` |
| `POST` | `/items/batch/quickmatch` | `LibraryItemController.batchQuickMatch` |  | `true` |
| `POST` | `/items/batch/scan` | `LibraryItemController.batchScan` |  | `true` |
| `POST` | `/items/batch/update` | `LibraryItemController.batchUpdate` |  | `true` |
| `GET` | `/^\/libraries/` | `` | `this.apiCacheManager.middleware` | `false` |
| `GET` | `/libraries` | `LibraryController.findAll` |  | `false` |
| `POST` | `/libraries` | `LibraryController.create` |  | `true` |
| `DELETE` | `/libraries/:id` | `LibraryController.delete` | `LibraryController.middleware` | `true` |
| `GET` | `/libraries/:id` | `LibraryController.findOne` | `LibraryController.middleware` | `false` |
| `PATCH` | `/libraries/:id` | `LibraryController.update` | `LibraryController.middleware` | `true` |
| `GET` | `/libraries/:id/authors` | `LibraryController.getAuthors` | `LibraryController.middleware` | `false` |
| `GET` | `/libraries/:id/collections` | `LibraryController.getCollectionsForLibrary` | `LibraryController.middleware` | `false` |
| `GET` | `/libraries/:id/download` | `LibraryController.downloadMultiple` | `LibraryController.middleware` | `false` |
| `GET` | `/libraries/:id/episode-downloads` | `LibraryController.getEpisodeDownloadQueue` | `LibraryController.middleware` | `false` |
| `GET` | `/libraries/:id/filterdata` | `LibraryController.getLibraryFilterData` | `LibraryController.middleware` | `false` |
| `DELETE` | `/libraries/:id/issues` | `LibraryController.removeLibraryItemsWithIssues` | `LibraryController.middleware` | `true` |
| `GET` | `/libraries/:id/items` | `LibraryController.getLibraryItems` | `LibraryController.middleware` | `false` |
| `GET` | `/libraries/:id/matchall` | `LibraryController.matchAll` | `LibraryController.middleware` | `false` |
| `GET` | `/libraries/:id/narrators` | `LibraryController.getNarrators` | `LibraryController.middleware` | `false` |
| `DELETE` | `/libraries/:id/narrators/:narratorId` | `LibraryController.removeNarrator` | `LibraryController.middleware` | `true` |
| `PATCH` | `/libraries/:id/narrators/:narratorId` | `LibraryController.updateNarrator` | `LibraryController.middleware` | `true` |
| `GET` | `/libraries/:id/opml` | `LibraryController.getOPMLFile` | `LibraryController.middleware` | `false` |
| `GET` | `/libraries/:id/personalized` | `LibraryController.getUserPersonalizedShelves` | `LibraryController.middleware` | `false` |
| `GET` | `/libraries/:id/playlists` | `LibraryController.getUserPlaylistsForLibrary` | `LibraryController.middleware` | `false` |
| `GET` | `/libraries/:id/podcast-titles` | `LibraryController.getPodcastTitles` | `LibraryController.middleware` | `false` |
| `GET` | `/libraries/:id/recent-episodes` | `LibraryController.getRecentEpisodes` | `LibraryController.middleware` | `false` |
| `POST` | `/libraries/:id/remove-metadata` | `LibraryController.removeAllMetadataFiles` | `LibraryController.middleware` | `true` |
| `POST` | `/libraries/:id/scan` | `LibraryController.scan` | `LibraryController.middleware` | `true` |
| `GET` | `/libraries/:id/search` | `LibraryController.search` | `LibraryController.middleware` | `false` |
| `GET` | `/libraries/:id/series` | `LibraryController.getAllSeriesForLibrary` | `LibraryController.middleware` | `false` |
| `GET` | `/libraries/:id/series/:seriesId` | `LibraryController.getSeriesForLibrary` | `LibraryController.middleware` | `false` |
| `GET` | `/libraries/:id/stats` | `LibraryController.stats` | `LibraryController.middleware` | `false` |
| `POST` | `/libraries/order` | `LibraryController.reorder` |  | `true` |
| `GET` | `/logger-data` | `MiscController.getLoggerData` |  | `false` |
| `GET` | `/me` | `MeController.getCurrentUser` |  | `false` |
| `POST` | `/me/ereader-devices` | `MeController.updateUserEReaderDevices` |  | `true` |
| `PATCH` | `/me/item/:id/bookmark` | `MeController.updateBookmark` |  | `true` |
| `POST` | `/me/item/:id/bookmark` | `MeController.createBookmark` |  | `true` |
| `DELETE` | `/me/item/:id/bookmark/:time` | `MeController.removeBookmark` |  | `true` |
| `GET` | `/me/item/listening-sessions/:libraryItemId/:episodeId?` | `MeController.getItemListeningSessions` |  | `false` |
| `GET` | `/me/items-in-progress` | `MeController.getAllLibraryItemsInProgress` |  | `false` |
| `GET` | `/me/listening-sessions` | `MeController.getListeningSessions` |  | `false` |
| `GET` | `/me/listening-stats` | `MeController.getListeningStats` |  | `false` |
| `PATCH` | `/me/password` | `MeController.updatePassword` | `this.auth.authRateLimiter` | `true` |
| `DELETE` | `/me/progress/:id` | `MeController.removeMediaProgress` |  | `true` |
| `GET` | `/me/progress/:id/:episodeId?` | `MeController.getMediaProgress` |  | `false` |
| `GET` | `/me/progress/:id/remove-from-continue-listening` | `MeController.removeItemFromContinueListening` |  | `false` |
| `PATCH` | `/me/progress/:libraryItemId/:episodeId?` | `MeController.createUpdateMediaProgress` |  | `true` |
| `PATCH` | `/me/progress/batch/update` | `MeController.batchUpdateMediaProgress` |  | `true` |
| `GET` | `/me/series/:id/readd-to-continue-listening` | `MeController.readdSeriesFromContinueListening` |  | `false` |
| `GET` | `/me/series/:id/remove-from-continue-listening` | `MeController.removeSeriesFromContinueListening` |  | `false` |
| `GET` | `/me/stats/year/:year` | `MeController.getStatsForYear` |  | `false` |
| `GET` | `/notificationdata` | `NotificationController.getData` | `NotificationController.middleware` | `false` |
| `GET` | `/notifications` | `NotificationController.get` | `NotificationController.middleware` | `false` |
| `PATCH` | `/notifications` | `NotificationController.update` | `NotificationController.middleware` | `true` |
| `POST` | `/notifications` | `NotificationController.createNotification` | `NotificationController.middleware` | `true` |
| `DELETE` | `/notifications/:id` | `NotificationController.deleteNotification` | `NotificationController.middleware` | `true` |
| `PATCH` | `/notifications/:id` | `NotificationController.updateNotification` | `NotificationController.middleware` | `true` |
| `GET` | `/notifications/:id/test` | `NotificationController.sendNotificationTest` | `NotificationController.middleware` | `false` |
| `GET` | `/notifications/test` | `NotificationController.fireTestEvent` | `NotificationController.middleware` | `false` |
| `GET` | `/playlists` | `PlaylistController.findAllForUser` |  | `false` |
| `POST` | `/playlists` | `PlaylistController.create` |  | `true` |
| `DELETE` | `/playlists/:id` | `PlaylistController.delete` | `PlaylistController.middleware` | `true` |
| `GET` | `/playlists/:id` | `PlaylistController.findOne` | `PlaylistController.middleware` | `false` |
| `PATCH` | `/playlists/:id` | `PlaylistController.update` | `PlaylistController.middleware` | `true` |
| `POST` | `/playlists/:id/batch/add` | `PlaylistController.addBatch` | `PlaylistController.middleware` | `true` |
| `POST` | `/playlists/:id/batch/remove` | `PlaylistController.removeBatch` | `PlaylistController.middleware` | `true` |
| `POST` | `/playlists/:id/item` | `PlaylistController.addItem` | `PlaylistController.middleware` | `true` |
| `DELETE` | `/playlists/:id/item/:libraryItemId/:episodeId?` | `PlaylistController.removeItem` | `PlaylistController.middleware` | `true` |
| `POST` | `/playlists/collection/:collectionId` | `PlaylistController.createFromCollection` |  | `true` |
| `POST` | `/podcasts` | `PodcastController.create` |  | `true` |
| `GET` | `/podcasts/:id/checknew` | `PodcastController.checkNewEpisodes` | `PodcastController.middleware` | `false` |
| `GET` | `/podcasts/:id/clear-queue` | `PodcastController.clearEpisodeDownloadQueue` | `PodcastController.middleware` | `false` |
| `POST` | `/podcasts/:id/download-episodes` | `PodcastController.downloadEpisodes` | `PodcastController.middleware` | `true` |
| `GET` | `/podcasts/:id/downloads` | `PodcastController.getEpisodeDownloads` | `PodcastController.middleware` | `false` |
| `DELETE` | `/podcasts/:id/episode/:episodeId` | `PodcastController.removeEpisode` | `PodcastController.middleware` | `true` |
| `GET` | `/podcasts/:id/episode/:episodeId` | `PodcastController.getEpisode` | `PodcastController.middleware` | `false` |
| `PATCH` | `/podcasts/:id/episode/:episodeId` | `PodcastController.updateEpisode` | `PodcastController.middleware` | `true` |
| `POST` | `/podcasts/:id/match-episodes` | `PodcastController.quickMatchEpisodes` | `PodcastController.middleware` | `true` |
| `GET` | `/podcasts/:id/search-episode` | `PodcastController.findEpisode` | `PodcastController.middleware` | `false` |
| `POST` | `/podcasts/feed` | `PodcastController.getPodcastFeed` |  | `true` |
| `POST` | `/podcasts/opml/create` | `PodcastController.bulkCreatePodcastsFromOpmlFeedUrls` |  | `true` |
| `POST` | `/podcasts/opml/parse` | `PodcastController.getFeedsFromOPMLText` |  | `true` |
| `GET` | `/search/authors` | `SearchController.findAuthor` |  | `false` |
| `GET` | `/search/books` | `SearchController.findBooks` |  | `false` |
| `GET` | `/search/chapters` | `SearchController.findChapters` |  | `false` |
| `GET` | `/search/covers` | `SearchController.findCovers` |  | `false` |
| `GET` | `/search/podcast` | `SearchController.findPodcasts` |  | `false` |
| `GET` | `/search/providers` | `SearchController.getAllProviders` |  | `false` |
| `GET` | `/series/:id` | `SeriesController.findOne` | `SeriesController.middleware` | `false` |
| `PATCH` | `/series/:id` | `SeriesController.update` | `SeriesController.middleware` | `true` |
| `GET` | `/session/:id` | `SessionController.getOpenSession` | `SessionController.openSessionMiddleware` | `false` |
| `POST` | `/session/:id/close` | `SessionController.close` | `SessionController.openSessionMiddleware` | `true` |
| `POST` | `/session/:id/sync` | `SessionController.sync` | `SessionController.openSessionMiddleware` | `true` |
| `POST` | `/session/local` | `SessionController.syncLocal` |  | `true` |
| `POST` | `/session/local-all` | `SessionController.syncLocalSessions` |  | `true` |
| `GET` | `/sessions` | `SessionController.getAllWithUserData` |  | `false` |
| `DELETE` | `/sessions/:id` | `SessionController.delete` | `SessionController.middleware` | `true` |
| `POST` | `/sessions/batch/delete` | `SessionController.batchDelete` |  | `true` |
| `GET` | `/sessions/open` | `SessionController.getOpenSessions` |  | `false` |
| `PATCH` | `/settings` | `MiscController.updateServerSettings` |  | `true` |
| `POST` | `/share/mediaitem` | `ShareController.createMediaItemShare` |  | `true` |
| `DELETE` | `/share/mediaitem/:id` | `ShareController.deleteMediaItemShare` |  | `true` |
| `PATCH` | `/sorting-prefixes` | `MiscController.updateSortingPrefixes` |  | `true` |
| `GET` | `/stats/server` | `StatsController.getServerStats` | `StatsController.middleware` | `false` |
| `GET` | `/stats/year/:year` | `StatsController.getAdminStatsForYear` | `StatsController.middleware` | `false` |
| `GET` | `/tags` | `MiscController.getAllTags` |  | `false` |
| `DELETE` | `/tags/:tag` | `MiscController.deleteTag` |  | `true` |
| `POST` | `/tags/rename` | `MiscController.renameTag` |  | `true` |
| `GET` | `/tasks` | `MiscController.getTasks` |  | `false` |
| `POST` | `/tools/batch/embed-metadata` | `ToolsController.batchEmbedMetadata` | `ToolsController.middleware` | `true` |
| `POST` | `/tools/item/:id/embed-metadata` | `ToolsController.embedAudioFileMetadata` | `ToolsController.middleware` | `true` |
| `DELETE` | `/tools/item/:id/encode-m4b` | `ToolsController.cancelM4bEncode` | `ToolsController.middleware` | `true` |
| `POST` | `/tools/item/:id/encode-m4b` | `ToolsController.encodeM4b` | `ToolsController.middleware` | `true` |
| `POST` | `/upload` | `MiscController.handleUpload` |  | `true` |
| `GET` | `/users` | `UserController.findAll` | `UserController.middleware` | `false` |
| `POST` | `/users` | `UserController.create` | `UserController.middleware` | `true` |
| `DELETE` | `/users/:id` | `UserController.delete` | `UserController.middleware` | `true` |
| `GET` | `/users/:id` | `UserController.findOne` | `UserController.middleware` | `false` |
| `PATCH` | `/users/:id` | `UserController.update` | `UserController.middleware` | `true` |
| `GET` | `/users/:id/listening-sessions` | `UserController.getListeningSessions` | `UserController.middleware` | `false` |
| `GET` | `/users/:id/listening-stats` | `UserController.getListeningStats` | `UserController.middleware` | `false` |
| `PATCH` | `/users/:id/openid-unlink` | `UserController.unlinkFromOpenID` | `UserController.middleware` | `true` |
| `GET` | `/users/online` | `UserController.getOnlineUsers` |  | `false` |
| `POST` | `/validate-cron` | `MiscController.validateCronExpression` |  | `true` |
| `POST` | `/watcher/update` | `MiscController.updateWatchedPath` |  | `true` |
