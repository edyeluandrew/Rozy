class Place {
  const Place({
    required this.id,
    required this.name,
    required this.lat,
    required this.lng,
    this.landmarkNote,
  });

  final String id;
  final String name;
  final double lat;
  final double lng;
  final String? landmarkNote;

  factory Place.fromJson(Map<String, dynamic> json) {
    return Place(
      id: json['id'] as String,
      name: json['name'] as String,
      lat: (json['lat'] as num).toDouble(),
      lng: (json['lng'] as num).toDouble(),
      landmarkNote: json['landmark_note'] as String?,
    );
  }
}
