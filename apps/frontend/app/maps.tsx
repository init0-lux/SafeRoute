import React, { useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, Dimensions } from 'react-native';
import { router } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import SearchInput from '@/components/SearchInput';

// PLACEHOLDER: Replace with your actual Google Maps API key
// For Android: Add to app.json under android.config.googleMaps.apiKey
// For iOS: Add to app.json under ios.config.googleMapsApiKey
const GOOGLE_MAPS_API_KEY = 'YOUR_API_KEY_HERE';

const { width, height } = Dimensions.get('window');

// Graceful map import
let MapView: React.ComponentType<any> | null = null;
let Marker: React.ComponentType<any> | null = null;
try {
    const maps = require('react-native-maps');
    MapView = maps.default;
    Marker = maps.Marker;
} catch {
    MapView = null;
    Marker = null;
}

export default function MapsScreen() {
    const [searchText, setSearchText] = useState('');
    const [mapError, setMapError] = useState(false);

    return (
        <View style={styles.container}>
            {/* Full-screen map background */}
            {MapView && !mapError ? (
                <MapView
                    style={StyleSheet.absoluteFillObject}
                    initialRegion={{
                        latitude: 33.5186,
                        longitude: -86.8104,
                        latitudeDelta: 0.05,
                        longitudeDelta: 0.05,
                    }}
                    onError={() => setMapError(true)}
                    showsUserLocation
                    showsMyLocationButton={false}
                />
            ) : (
                <View style={styles.mapFallback}>
                    <Ionicons name="map-outline" size={56} color={Colors.gray} />
                    <Text style={styles.mapFallbackTitle}>Map could not be loaded</Text>
                    <Text style={styles.mapFallbackSub}>
                        Check your API key in constants/config.ts{'\n'}
                        The app continues to work without maps
                    </Text>
                </View>
            )}

            {/* Back button */}
            <TouchableOpacity
                style={styles.backButton}
                onPress={() => router.back()}
                activeOpacity={0.8}
            >
                <Ionicons name="arrow-back" size={22} color={Colors.bgDark} />
            </TouchableOpacity>

            {/* Search overlay at top */}
            <View style={styles.searchOverlay}>
                <SearchInput
                    placeholder="Search location..."
                    value={searchText}
                    onChangeText={setSearchText}
                />
            </View>

            {/* Bottom info card */}
            <View style={styles.bottomCard}>
                <View style={styles.handle} />
                <Text style={styles.locationTitle}>Your Location</Text>
                <Text style={styles.locationSub}>Sharing your live location</Text>

                <View style={styles.row}>
                    <View style={styles.coordBox}>
                        <Ionicons name="location" size={16} color={Colors.purple} />
                        <Text style={styles.coordText}>33.5186° N, 86.8104° W</Text>
                    </View>
                </View>

                <View style={styles.actionsRow}>
                    <TouchableOpacity style={styles.actionButton} activeOpacity={0.7}>
                        <Ionicons name="navigate" size={20} color={Colors.white} />
                        <Text style={styles.actionText}>Directions</Text>
                    </TouchableOpacity>
                    <TouchableOpacity
                        style={[styles.actionButton, styles.shareBtn]}
                        activeOpacity={0.7}
                    >
                        <Ionicons name="share-outline" size={20} color={Colors.bgDark} />
                        <Text style={[styles.actionText, styles.shareBtnText]}>Share</Text>
                    </TouchableOpacity>
                </View>
            </View>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: Colors.bgDark,
    },
    mapFallback: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        backgroundColor: Colors.cardBg,
        gap: Spacing.md,
        paddingHorizontal: Spacing.xl,
    },
    mapFallbackTitle: {
        fontSize: FontSizes.xl,
        fontWeight: '700',
        color: Colors.grayLight,
        textAlign: 'center',
    },
    mapFallbackSub: {
        fontSize: FontSizes.sm,
        color: Colors.gray,
        textAlign: 'center',
        lineHeight: FontSizes.sm * 1.6,
    },
    backButton: {
        position: 'absolute',
        top: 56,
        left: Spacing.lg,
        width: 40,
        height: 40,
        borderRadius: 20,
        backgroundColor: Colors.white,
        justifyContent: 'center',
        alignItems: 'center',
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 2 },
        shadowOpacity: 0.15,
        shadowRadius: 4,
        elevation: 5,
        zIndex: 20,
    },
    searchOverlay: {
        position: 'absolute',
        top: 56,
        left: 70,
        right: Spacing.lg,
        zIndex: 10,
    },
    bottomCard: {
        position: 'absolute',
        bottom: 0,
        left: 0,
        right: 0,
        backgroundColor: Colors.cardBg,
        borderTopLeftRadius: BorderRadius.xl,
        borderTopRightRadius: BorderRadius.xl,
        paddingHorizontal: Spacing.lg,
        paddingTop: Spacing.md,
        paddingBottom: 40,
    },
    handle: {
        width: 40,
        height: 4,
        borderRadius: 2,
        backgroundColor: Colors.grayDark,
        alignSelf: 'center',
        marginBottom: Spacing.lg,
    },
    locationTitle: {
        fontSize: FontSizes.lg,
        fontWeight: '700',
        color: Colors.white,
        marginBottom: 4,
    },
    locationSub: {
        fontSize: FontSizes.sm,
        color: Colors.grayLight,
        marginBottom: Spacing.lg,
    },
    row: {
        marginBottom: Spacing.lg,
    },
    coordBox: {
        flexDirection: 'row',
        alignItems: 'center',
        gap: Spacing.sm,
        backgroundColor: Colors.cardBgLight,
        paddingHorizontal: Spacing.md,
        paddingVertical: Spacing.sm,
        borderRadius: BorderRadius.sm,
    },
    coordText: {
        fontSize: FontSizes.sm,
        color: Colors.grayLight,
        fontFamily: 'monospace',
    },
    actionsRow: {
        flexDirection: 'row',
        gap: Spacing.md,
    },
    actionButton: {
        flex: 1,
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        gap: Spacing.sm,
        backgroundColor: Colors.cardBgLight,
        paddingVertical: Spacing.md,
        borderRadius: BorderRadius.md,
        borderWidth: 1,
        borderColor: Colors.border,
    },
    actionText: {
        fontSize: FontSizes.sm,
        fontWeight: '600',
        color: Colors.white,
    },
    shareBtn: {
        backgroundColor: Colors.purple,
        borderColor: Colors.purple,
    },
    shareBtnText: {
        color: Colors.bgDark,
    },
});
