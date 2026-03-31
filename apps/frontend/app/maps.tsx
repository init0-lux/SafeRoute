import React, { useState, useEffect, useRef } from 'react';
import {
    View,
    Text,
    StyleSheet,
    TouchableOpacity,
    TextInput,
    ActivityIndicator,
    Dimensions,
    Platform,
    FlatList,
    Keyboard,
} from 'react-native';
import { router } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import BottomNavBar from '@/components/BottomNavBar';
import * as Location from 'expo-location';
import {
    getRouteSafetyScore,
    searchPlaces,
    getPlaceDetails,
    PlacePrediction,
    Coordinates,
    RouteSafetyResponse,
} from '@/services/safety';

let MapView: any = null;
let Marker: any = null;
let Polyline: any = null;
let PROVIDER_GOOGLE: any = null;
try {
    const maps = require('react-native-maps');
    MapView = maps.default;
    Marker = maps.Marker;
    Polyline = maps.Polyline;
    PROVIDER_GOOGLE = maps.PROVIDER_GOOGLE;
} catch {
    MapView = null;
    Marker = null;
    Polyline = null;
}

const { width, height } = Dimensions.get('window');

// Light mode map style to match mockup
const lightMapStyle = [
    {
        "featureType": "administrative",
        "elementType": "geometry",
        "stylers": [{ "visibility": "off" }]
    },
    {
        "featureType": "poi",
        "stylers": [{ "visibility": "off" }]
    },
    {
        "featureType": "road",
        "elementType": "labels.icon",
        "stylers": [{ "visibility": "off" }]
    },
    {
        "featureType": "transit",
        "stylers": [{ "visibility": "off" }]
    }
];

export default function MapsScreen() {
    const [fromText, setFromText] = useState('My Current Location');
    const [toText, setToText] = useState('');
    const [location, setLocation] = useState<Location.LocationObject | null>(null);
    const [locationLoading, setLocationLoading] = useState(true);
    const [routeData, setRouteData] = useState<RouteSafetyResponse | null>(null);
    const [isRouting, setIsRouting] = useState(false);
    const [destination, setDestination] = useState<Coordinates | null>(null);
    
    // Autocomplete state
    const [predictions, setPredictions] = useState<PlacePrediction[]>([]);
    const [showPredictions, setShowPredictions] = useState(false);
    const [isSearching, setIsSearching] = useState(false);
    const searchTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
    
    const mapRef = useRef<any>(null);

    useEffect(() => {
        (async () => {
            const { status } = await Location.requestForegroundPermissionsAsync();
            if (status !== 'granted') {
                setLocationLoading(false);
                return;
            }

            const curr = await Location.getCurrentPositionAsync({});
            setLocation(curr);
            setLocationLoading(false);
        })();
    }, []);

    const onTabPress = (tab: string) => {
        if (tab === 'sos') {
            router.push('/sos' as any);
        } else if (tab === 'home') {
            router.push('/home' as any);
        } else if (tab === 'history') {
            router.push('/history' as any);
        }
    };

    // Debounced place search
    const handleDestinationChange = (text: string) => {
        setToText(text);
        
        if (searchTimeout.current) {
            clearTimeout(searchTimeout.current);
        }

        if (text.trim().length < 2) {
            setPredictions([]);
            setShowPredictions(false);
            return;
        }

        searchTimeout.current = setTimeout(async () => {
            setIsSearching(true);
            const userLocation = location ? {
                lat: location.coords.latitude,
                lng: location.coords.longitude,
            } : undefined;
            
            const results = await searchPlaces(text, userLocation);
            setPredictions(results);
            setShowPredictions(results.length > 0);
            setIsSearching(false);
        }, 300);
    };

    const handleSelectPlace = async (prediction: PlacePrediction) => {
        Keyboard.dismiss();
        setShowPredictions(false);
        setToText(prediction.description);
        setIsRouting(true);

        try {
            const placeDetails = await getPlaceDetails(prediction.place_id);
            if (!placeDetails || !location) {
                throw new Error('Could not get place details');
            }

            setDestination(placeDetails.coordinates);
            
            const data = await getRouteSafetyScore(
                {
                    lat: location.coords.latitude,
                    lng: location.coords.longitude,
                },
                placeDetails.coordinates,
                'walking'
            );

            setRouteData(data);
            
            // Zoom to show the route
            if (mapRef.current) {
                mapRef.current.fitToCoordinates([
                    { latitude: location.coords.latitude, longitude: location.coords.longitude },
                    { latitude: placeDetails.coordinates.lat, longitude: placeDetails.coordinates.lng }
                ], {
                    edgePadding: { top: 150, right: 50, bottom: 350, left: 50 },
                    animated: true
                });
            }
        } catch (err) {
            console.error('Routing failed:', err);
        } finally {
            setIsRouting(false);
        }
    };

    const handleSearch = async () => {
        if (!toText || !location) return;
        
        // If user typed without selecting from autocomplete, try to search and use first result
        if (predictions.length > 0) {
            await handleSelectPlace(predictions[0]);
        } else {
            // Trigger a search
            const userLocation = {
                lat: location.coords.latitude,
                lng: location.coords.longitude,
            };
            const results = await searchPlaces(toText, userLocation);
            if (results.length > 0) {
                await handleSelectPlace(results[0]);
            }
        }
    };

    const renderPredictionItem = ({ item }: { item: PlacePrediction }) => (
        <TouchableOpacity
            style={styles.predictionItem}
            onPress={() => handleSelectPlace(item)}
            activeOpacity={0.7}
        >
            <Ionicons name="location-outline" size={18} color={Colors.gray} style={{ marginRight: 10 }} />
            <View style={{ flex: 1 }}>
                <Text style={styles.predictionMain} numberOfLines={1}>{item.main_text}</Text>
                <Text style={styles.predictionSecondary} numberOfLines={1}>{item.secondary_text}</Text>
            </View>
        </TouchableOpacity>
    );

    return (
        <View style={styles.container}>
            {MapView && location ? (
                <MapView
                    ref={mapRef}
                    provider={PROVIDER_GOOGLE}
                    style={StyleSheet.absoluteFillObject}
                    customMapStyle={lightMapStyle}
                    initialRegion={{
                        latitude: location.coords.latitude,
                        longitude: location.coords.longitude,
                        latitudeDelta: 0.02,
                        longitudeDelta: 0.02,
                    }}
                >
                    <Marker
                        coordinate={{
                            latitude: location.coords.latitude,
                            longitude: location.coords.longitude,
                        }}
                        title="You"
                    >
                        <View style={styles.userMarker}>
                            <View style={styles.userMarkerInner} />
                        </View>
                    </Marker>

                    {routeData && (
                        <Polyline
                            coordinates={decodePolyline(routeData.route.polyline)}
                            strokeWidth={6}
                            strokeColor={Colors.bgDark}
                        />
                    )}

                    {destination && (
                        <Marker
                            coordinate={{
                                latitude: destination.lat,
                                longitude: destination.lng,
                            }}
                            title="Destination"
                        >
                            <View style={styles.destinationMarker}>
                                <Ionicons name="location" size={28} color={Colors.sosRed} />
                            </View>
                        </Marker>
                    )}
                </MapView>
            ) : (
                <View style={styles.center}>
                    <ActivityIndicator size="large" color={Colors.bgDark} />
                </View>
            )}

            {/* Back Button */}
            <TouchableOpacity
                style={styles.backButton}
                onPress={() => router.back()}
            >
                <Ionicons name="arrow-back" size={24} color={Colors.bgDark} />
            </TouchableOpacity>

            {/* Destination Overlays */}
            <View style={styles.searchOverlay}>
                <View style={styles.searchInputWrapper}>
                    <Text style={styles.searchLabel}>From:</Text>
                    <TextInput
                        style={styles.searchInput}
                        value={fromText}
                        onChangeText={setFromText}
                        placeholder="Current Location"
                        editable={false}
                    />
                </View>
                <View style={styles.searchInputWrapper}>
                    <Text style={styles.searchLabel}>To:</Text>
                    <TextInput
                        style={styles.searchInput}
                        value={toText}
                        onChangeText={handleDestinationChange}
                        placeholder="Enter destination"
                        onSubmitEditing={handleSearch}
                        returnKeyType="search"
                        onFocus={() => predictions.length > 0 && setShowPredictions(true)}
                    />
                    {(isRouting || isSearching) && <ActivityIndicator size="small" color={Colors.bgDark} />}
                </View>
                
                {/* Autocomplete Predictions Dropdown */}
                {showPredictions && (
                    <View style={styles.predictionsContainer}>
                        <FlatList
                            data={predictions}
                            keyExtractor={(item) => item.place_id}
                            renderItem={renderPredictionItem}
                            keyboardShouldPersistTaps="handled"
                            showsVerticalScrollIndicator={false}
                        />
                    </View>
                )}
            </View>

            {/* Quick Report FAB */}
            <TouchableOpacity 
                style={styles.quickReportFab} 
                onPress={() => router.push('/home' as any)}
            >
                <Text style={styles.fabText}>quick report</Text>
                <View style={styles.fabIcon}>
                    <Ionicons name="add" size={20} color={Colors.white} />
                </View>
            </TouchableOpacity>

            {/* Safest Route Bottom Sheet */}
            <View style={styles.bottomSheet}>
                <View style={styles.dragHandle} />
                <Text style={styles.sheetTitle}>Safest Route</Text>
                
                <View style={styles.metricsContainer}>
                    <View style={styles.metricRow}>
                        <Ionicons name="shield-checkmark" size={18} color={Colors.success} />
                        <Text style={styles.metricText}>safest route evaluated</Text>
                    </View>

                    <View style={styles.scoreRow}>
                        <View>
                            <Text style={styles.scoreLabel}>safety score</Text>
                            <Text style={styles.scoreValue}>{routeData?.score ?? '--'}</Text>
                        </View>
                        <View style={styles.divider} />
                        <View>
                            <Text style={styles.scoreLabel}>incidents reported</Text>
                            <Text style={styles.scoreValue}>{routeData?.summary.recent_reports ?? '--'}</Text>
                        </View>
                    </View>

                    <View style={styles.tipsSection}>
                        <Text style={styles.tipsLabel}>Safety Tips:</Text>
                        <Text style={styles.tipsValue}>
                            {routeData?.score && routeData.score > 80 
                                ? "This route is well-lit and highly trafficked. Stay alert but proceed with confidence."
                                : "Expect some dimly lit areas. We recommend staying on the main road and keeping your SOS shortcut ready."}
                        </Text>
                    </View>
                </View>
            </View>

            <View style={styles.bottomNav}>
                <BottomNavBar activeTab="location" onTabPress={onTabPress} />
            </View>
        </View>
    );
}

// Simple polyline decoder
function decodePolyline(encoded: string) {
    if (!encoded) return [];
    var points = []
    var index = 0, len = encoded.length;
    var lat = 0, lng = 0;
    while (index < len) {
        var b, shift = 0, result = 0;
        do {
            b = encoded.charCodeAt(index++) - 63;
            result |= (b & 0x1f) << shift;
            shift += 5;
        } while (b >= 0x20);
        var dlat = ((result & 1) ? ~(result >> 1) : (result >> 1));
        lat += dlat;
        shift = 0;
        result = 0;
        do {
            b = encoded.charCodeAt(index++) - 63;
            result |= (b & 0x1f) << shift;
            shift += 5;
        } while (b >= 0x20);
        var dlng = ((result & 1) ? ~(result >> 1) : (result >> 1));
        lng += dlng;
        points.push({ latitude: (lat / 1e5), longitude: (lng / 1e5) });
    }
    return points;
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: Colors.white,
    },
    center: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
    },
    userMarker: {
        width: 24,
        height: 24,
        borderRadius: 12,
        backgroundColor: 'rgba(108, 92, 231, 0.2)',
        justifyContent: 'center',
        alignItems: 'center',
    },
    userMarkerInner: {
        width: 12,
        height: 12,
        borderRadius: 6,
        backgroundColor: Colors.purple,
        borderWidth: 2,
        borderColor: Colors.white,
    },
    backButton: {
        position: 'absolute',
        top: 60,
        left: Spacing.lg,
        width: 44,
        height: 44,
        borderRadius: 22,
        backgroundColor: Colors.white,
        borderWidth: 3,
        borderColor: Colors.bgDark,
        justifyContent: 'center',
        alignItems: 'center',
        zIndex: 20,
    },
    searchOverlay: {
        position: 'absolute',
        top: 60,
        left: 80,
        right: Spacing.lg,
        zIndex: 10,
        gap: 10,
    },
    searchInputWrapper: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: Colors.white,
        borderWidth: 3,
        borderColor: Colors.bgDark,
        borderRadius: BorderRadius.full,
        paddingHorizontal: 16,
        paddingVertical: 10,
    },
    searchLabel: {
        fontSize: 12,
        fontWeight: '800',
        color: Colors.gray,
        marginRight: 6,
    },
    searchInput: {
        flex: 1,
        fontSize: 14,
        fontWeight: '700',
        color: Colors.bgDark,
    },
    quickReportFab: {
        position: 'absolute',
        bottom: 310,
        right: Spacing.lg,
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: Colors.white,
        borderWidth: 3,
        borderColor: Colors.bgDark,
        borderRadius: BorderRadius.full,
        paddingLeft: 16,
        paddingRight: 6,
        paddingVertical: 6,
        gap: 10,
        zIndex: 30,
        shadowColor: Colors.bgDark,
        shadowOffset: { width: 4, height: 4 },
        shadowOpacity: 0.2,
        shadowRadius: 0,
    },
    fabText: {
        fontSize: 12,
        fontWeight: '900',
        color: Colors.bgDark,
    },
    fabIcon: {
        width: 28,
        height: 28,
        borderRadius: 14,
        backgroundColor: Colors.bgDark,
        justifyContent: 'center',
        alignItems: 'center',
    },
    bottomSheet: {
        position: 'absolute',
        bottom: -20,
        left: 0,
        right: 0,
        backgroundColor: '#333333',
        borderTopLeftRadius: 32,
        borderTopRightRadius: 32,
        paddingHorizontal: Spacing.xl,
        paddingTop: 12,
        paddingBottom: 120,
        zIndex: 10,
    },
    dragHandle: {
        width: 40,
        height: 4,
        backgroundColor: 'rgba(255,255,255,0.2)',
        borderRadius: 2,
        alignSelf: 'center',
        marginBottom: 20,
    },
    sheetTitle: {
        fontSize: 24,
        fontWeight: '900',
        color: Colors.white,
        marginBottom: 20,
    },
    metricsContainer: {
        gap: 20,
    },
    metricRow: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: 8,
    },
    metricText: {
        fontSize: 14,
        fontWeight: '600',
        color: Colors.success,
    },
    scoreRow: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: 'rgba(0,0,0,0.2)',
        borderRadius: 20,
        padding: 20,
    },
    scoreLabel: {
        fontSize: 10,
        fontWeight: '700',
        color: Colors.grayLight,
        textTransform: 'uppercase',
        marginBottom: 4,
    },
    scoreValue: {
        fontSize: 24,
        fontWeight: '900',
        color: Colors.white,
    },
    divider: {
        width: 1,
        height: '100%',
        backgroundColor: 'rgba(255,255,255,0.1)',
        marginHorizontal: 30,
    },
    tipsSection: {
        gap: 8,
    },
    tipsLabel: {
        fontSize: 12,
        fontWeight: '800',
        color: Colors.grayLight,
    },
    tipsValue: {
        fontSize: 14,
        color: Colors.white,
        lineHeight: 20,
        opacity: 0.8,
    },
    bottomNav: {
        position: "absolute",
        bottom: 20,
        left: 0,
        right: 0,
        flexDirection: "row",
        justifyContent: "center",
        alignItems: "center",
        zIndex: 100,
    },
    destinationMarker: {
        alignItems: 'center',
        justifyContent: 'center',
    },
    predictionsContainer: {
        backgroundColor: Colors.white,
        borderWidth: 3,
        borderColor: Colors.bgDark,
        borderRadius: 16,
        marginTop: 4,
        maxHeight: 200,
        overflow: 'hidden',
    },
    predictionItem: {
        flexDirection: 'row',
        alignItems: 'center',
        paddingHorizontal: 16,
        paddingVertical: 12,
        borderBottomWidth: 1,
        borderBottomColor: 'rgba(0,0,0,0.05)',
    },
    predictionMain: {
        fontSize: 14,
        fontWeight: '700',
        color: Colors.bgDark,
    },
    predictionSecondary: {
        fontSize: 12,
        color: Colors.gray,
        marginTop: 2,
    },
});
