import React, { useState } from 'react';
import {
    View,
    Text,
    StyleSheet,
    TouchableOpacity,
    ScrollView,
    Dimensions,
} from 'react-native';
import { router } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import SearchInput from '@/components/SearchInput';
import BottomNavBar from '@/components/BottomNavBar';

const { width } = Dimensions.get('window');

// Try to import MapView gracefully
let MapView: React.ComponentType<any> | null = null;
try {
    MapView = require('react-native-maps').default;
} catch {
    MapView = null;
}

export default function HomeScreen() {
    const [destination, setDestination] = useState('');
    const [mapError, setMapError] = useState(false);
    const [activeTab, setActiveTab] = useState("home");

    const onTabPress = (tab: string) => {
        setActiveTab(tab);
    };


    const handleShareLocation = () => {
        router.push('/maps' as any);
    };

    return (
        <View style={styles.container}>
            <ScrollView
                style={styles.scrollView}
                contentContainerStyle={styles.scrollContent}
                showsVerticalScrollIndicator={false}
            >
                {/* Header: user greeting */}
                <View style={styles.header}>
                    <View style={styles.avatar}>
                        <Ionicons name="person" size={24} color={Colors.gray} />
                    </View>
                    <View style={styles.greeting}>
                        <Text style={styles.greetingName}>hello, user</Text>
                        <Text style={styles.greetingSub}>let us help you find the safest route</Text>
                    </View>
                </View>

                {/* Search destination */}
                <View style={styles.searchContainer}>
                    <SearchInput
                        placeholder="enter your destination"
                        value={destination}
                        onChangeText={setDestination}
                    />
                </View>

                {/* Map preview card */}
                <View style={styles.mapCard}>
                    {MapView && !mapError ? (
                        <View style={styles.mapWrapper}>
                            <MapView
                                style={styles.mapPreview}
                                initialRegion={{
                                    latitude: 33.5186,
                                    longitude: -86.8104,
                                    latitudeDelta: 0.03,
                                    longitudeDelta: 0.03,
                                }}
                                onError={() => setMapError(true)}
                                scrollEnabled={false}
                                zoomEnabled={false}
                                customMapStyle={darkMapStyle}
                            />
                            {/* YOU marker */}
                            <View style={styles.youMarker}>
                                <Text style={styles.youText}>YOU</Text>
                                <View style={styles.youDot} />
                            </View>
                        </View>
                    ) : (
                        <View style={styles.mapFallback}>
                            <Ionicons name="map-outline" size={40} color={Colors.gray} />
                            <Text style={styles.mapFallbackText}>
                                {mapError ? 'Map could not be loaded' : 'Maps not available'}
                            </Text>
                            <Text style={styles.mapFallbackSub}>
                                App continues to work without maps
                            </Text>
                        </View>
                    )}

                    {/* Share location button */}
                    <TouchableOpacity
                        style={styles.shareButton}
                        onPress={handleShareLocation}
                        activeOpacity={0.85}
                    >
                        <Text style={styles.shareText}>share location</Text>
                    </TouchableOpacity>
                </View>

                {/* Report Incident section - folder style */}
                <View style={styles.reportSection}>
                    <View style={styles.reportCard}>
                        <Text style={styles.reportTitle}>Report The Incident Here</Text>

                        {/* Nature of incident */}
                        <TouchableOpacity
                            style={styles.inputField}
                            onPress={() => router.push('/report' as any)}
                            activeOpacity={0.7}
                        >
                            <Text style={styles.inputPlaceholder}>
                                Nature of incident<Text style={styles.required}>*</Text>
                            </Text>
                            <Ionicons name="chevron-down" size={18} color={Colors.gray} />
                        </TouchableOpacity>

                        {/* Details */}
                        <View style={styles.detailsField}>
                            <Text style={styles.inputPlaceholder}>
                                provide details here<Text style={styles.required}>*</Text>
                            </Text>
                        </View>

                        {/* Upload evidence */}
                        <TouchableOpacity style={styles.uploadField} activeOpacity={0.7}>
                            <Text style={styles.uploadPlaceholder}>upload evidence</Text>
                            <Ionicons name="cloud-upload-outline" size={18} color={Colors.gray} />
                        </TouchableOpacity>

                        {/* Submit */}
                        <View style={styles.submitRow}>
                            <TouchableOpacity style={styles.submitButton} activeOpacity={0.85}>
                                <Text style={styles.submitText}>submit</Text>
                            </TouchableOpacity>
                        </View>
                    </View>
                </View>
            </ScrollView>
            <View style={styles.bottomNav}>
                <BottomNavBar activeTab={activeTab} onTabPress={onTabPress} />
            </View>
        </View>
    );
}

const darkMapStyle = [
    { elementType: 'geometry', stylers: [{ color: '#212121' }] },
    { elementType: 'labels.text.fill', stylers: [{ color: '#757575' }] },
    { elementType: 'labels.text.stroke', stylers: [{ color: '#212121' }] },
    { featureType: 'road', elementType: 'geometry', stylers: [{ color: '#2c2c2c' }] },
    { featureType: 'road', elementType: 'geometry.stroke', stylers: [{ color: '#212121' }] },
    { featureType: 'water', elementType: 'geometry', stylers: [{ color: '#000000' }] },
];

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: Colors.bgDark,
    },
    scrollView: {
        flex: 1,
    },
    scrollContent: {
        paddingBottom: 120,
    },
    header: {
        flexDirection: 'row',
        alignItems: 'center',
        paddingHorizontal: Spacing.lg,
        paddingTop: 60,
        paddingBottom: Spacing.lg,
        gap: Spacing.md,
    },
    avatar: {
        width: 48,
        height: 48,
        borderRadius: 24,
        backgroundColor: Colors.cardBg,
        justifyContent: 'center',
        alignItems: 'center',
    },
    greeting: {
        flex: 1,
    },
    greetingName: {
        fontSize: FontSizes.md,
        fontWeight: '700',
        color: Colors.white,
    },
    greetingSub: {
        fontSize: FontSizes.sm,
        color: Colors.grayLight,
        marginTop: 2,
    },
    searchContainer: {
        paddingHorizontal: Spacing.lg,
        marginBottom: Spacing.lg,
    },
    mapCard: {
        marginHorizontal: Spacing.lg,
        borderRadius: BorderRadius.lg,
        overflow: 'hidden',
        backgroundColor: Colors.cardBg,
        marginBottom: Spacing.lg,
    },
    mapWrapper: {
        height: 180,
        position: 'relative',
    },
    mapPreview: {
        flex: 1,
    },
    youMarker: {
        position: 'absolute',
        top: 20,
        left: 24,
        alignItems: 'center',
    },
    youText: {
        fontSize: 10,
        fontWeight: '800',
        color: Colors.white,
        backgroundColor: Colors.purple,
        paddingHorizontal: 8,
        paddingVertical: 2,
        borderRadius: 4,
        overflow: 'hidden',
        marginBottom: 4,
    },
    youDot: {
        width: 10,
        height: 10,
        borderRadius: 5,
        backgroundColor: Colors.white,
        borderWidth: 2,
        borderColor: Colors.bgDark,
    },
    mapFallback: {
        height: 180,
        justifyContent: 'center',
        alignItems: 'center',
        backgroundColor: Colors.cardBg,
        gap: Spacing.sm,
    },
    mapFallbackText: {
        fontSize: FontSizes.md,
        color: Colors.grayLight,
        fontWeight: '600',
    },
    mapFallbackSub: {
        fontSize: FontSizes.xs,
        color: Colors.gray,
    },
    shareButton: {
        backgroundColor: Colors.white,
        paddingVertical: Spacing.sm + 2,
        paddingHorizontal: Spacing.xl,
        borderRadius: BorderRadius.sm,
        alignSelf: 'flex-end',
        margin: Spacing.md,
        marginTop: -Spacing.xl,
        zIndex: 10,
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 2 },
        shadowOpacity: 0.15,
        shadowRadius: 4,
        elevation: 4,
    },
    shareText: {
        fontSize: FontSizes.sm,
        fontWeight: '600',
        color: Colors.bgDark,
    },
    reportSection: {
        marginHorizontal: Spacing.lg,
        marginTop: Spacing.md,
    },
    folderTab: {
        backgroundColor: Colors.purple,
        width: 40,
        height: 28,
        borderTopLeftRadius: BorderRadius.sm,
        borderTopRightRadius: BorderRadius.sm,
        justifyContent: 'center',
        alignItems: 'center',
        marginLeft: 20,
    },
    reportCard: {
        backgroundColor: Colors.white,
        borderRadius: BorderRadius.lg,
        padding: Spacing.lg,
        borderWidth: 1,
        borderColor: '#E0E0E0',
    },
    reportTitle: {
        fontSize: FontSizes.lg,
        fontWeight: '800',
        color: Colors.bgDark,
        marginBottom: Spacing.lg,
    },
    inputField: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        borderWidth: 1,
        borderColor: '#D0D0D0',
        borderRadius: BorderRadius.full,
        paddingHorizontal: Spacing.lg,
        paddingVertical: Spacing.md,
        marginBottom: Spacing.md,
    },
    inputPlaceholder: {
        fontSize: FontSizes.sm,
        color: Colors.gray,
    },
    required: {
        color: Colors.red,
    },
    detailsField: {
        borderWidth: 1,
        borderColor: '#D0D0D0',
        borderRadius: BorderRadius.md,
        paddingHorizontal: Spacing.lg,
        paddingVertical: Spacing.md,
        marginBottom: Spacing.md,
        minHeight: 100,
    },
    uploadField: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        borderWidth: 1,
        borderColor: '#D0D0D0',
        borderRadius: BorderRadius.md,
        paddingHorizontal: Spacing.lg,
        paddingVertical: Spacing.md,
        marginBottom: Spacing.lg,
    },
    uploadPlaceholder: {
        fontSize: FontSizes.sm,
        color: Colors.gray,
    },
    submitRow: {
        alignItems: 'center',
    },
    submitButton: {
        borderWidth: 2,
        borderColor: Colors.sosRed,
        borderRadius: BorderRadius.full,
        paddingHorizontal: Spacing.xxl,
        paddingVertical: Spacing.md,
        minWidth: 140,
        alignItems: 'center',
    },
    submitText: {
        fontSize: FontSizes.md,
        fontWeight: '700',
        color: Colors.bgDark,
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
});
