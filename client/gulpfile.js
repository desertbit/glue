'use strict';

var gulp 		   = require('gulp'),
	sourcemaps 	 = require('gulp-sourcemaps'),
	uglify 		    = require('gulp-uglify'),
	gulpif 		    = require('gulp-if'),
  fileinclude   = require('gulp-file-include');

var debug = false;


gulp.task('js', function () {
  gulp.src(['src/glue.js'])
    .pipe(fileinclude({
        prefix: '@@',
        basepath: '@file'
    }))
    .pipe(gulpif(debug, sourcemaps.init()))
      .pipe(gulpif(!debug, uglify()))
    .pipe(gulpif(debug, sourcemaps.write()))
    .pipe(gulp.dest('./dist/'));
})


gulp.task('watch', ['default'], function () {
  gulp.watch(['./src/*.js', './src/**/*.js'], ['js']);
});

gulp.task('setdebug', function() {
	debug = true;
});

gulp.task('debug', ['setdebug', 'default'], function() {

});

gulp.task('default', ['js'], function() {

});
