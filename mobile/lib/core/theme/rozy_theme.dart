import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

import 'rozy_colors.dart';

enum RozyAppVariant { passenger, driver }

abstract final class RozyTheme {
  static ThemeData forVariant(RozyAppVariant variant) {
    final isDriver = variant == RozyAppVariant.driver;

    final colorScheme = ColorScheme(
      brightness: Brightness.light,
      primary: RozyColors.gold,
      onPrimary: RozyColors.charcoal,
      secondary: RozyColors.darkGold,
      onSecondary: RozyColors.cream,
      error: const Color(0xFFB3261E),
      onError: RozyColors.white,
      surface: RozyColors.white,
      onSurface: RozyColors.charcoal,
    );

    final base = ThemeData(
      useMaterial3: true,
      colorScheme: colorScheme,
      scaffoldBackgroundColor: RozyColors.cream,
      cardColor: RozyColors.white,
      dividerColor: RozyColors.border,
      textTheme: GoogleFonts.interTextTheme().apply(
        bodyColor: RozyColors.charcoal,
        displayColor: RozyColors.charcoal,
      ),
      appBarTheme: AppBarTheme(
        backgroundColor: isDriver ? RozyColors.darkGold : RozyColors.cream,
        foregroundColor: isDriver ? RozyColors.cream : RozyColors.charcoal,
        elevation: 0,
        centerTitle: true,
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: RozyColors.gold,
          foregroundColor: RozyColors.charcoal,
          minimumSize: const Size.fromHeight(52),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
          ),
          textStyle: const TextStyle(fontWeight: FontWeight.w600, fontSize: 16),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: RozyColors.charcoal,
          side: const BorderSide(color: RozyColors.border),
          minimumSize: const Size.fromHeight(52),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
          ),
        ),
      ),
      cardTheme: CardThemeData(
        color: RozyColors.white,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
          side: const BorderSide(color: RozyColors.border, width: 0.5),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: RozyColors.white,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: RozyColors.border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: RozyColors.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: RozyColors.gold, width: 2),
        ),
      ),
    );

    return base;
  }

  /// Premium dark card used for wallet / earnings sections (driver app).
  static BoxDecoration premiumCardDecoration = BoxDecoration(
    color: RozyColors.black,
    borderRadius: BorderRadius.circular(16),
  );
}
